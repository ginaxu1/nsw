package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/LSFLK/argus/pkg/audit"
	"github.com/OpenNSW/nsw/internal/auth"
)

// HTTPHandler encapsulates the HTTP transport logic for TaskManager
type HTTPHandler struct {
	manager     TaskManager
	auditClient *audit.Client
}

// NewHTTPHandler creates a new HTTPHandler for the task manager
func NewHTTPHandler(manager TaskManager, auditClient *audit.Client) *HTTPHandler {
	return &HTTPHandler{
		manager:     manager,
		auditClient: auditClient,
	}
}

// HandleGetTask is an HTTP handler for fetching task information via GET request
func (h *HTTPHandler) HandleGetTask(w http.ResponseWriter, r *http.Request) {
	taskId := r.PathValue("id")
	if taskId == "" {
		writeJSONError(w, http.StatusBadRequest, "taskId is required")
		return
	}

	result, err := h.manager.GetTaskRenderInfo(r.Context(), taskId)

	// Fire the audit log asynchronously before returning the HTTP response.
	// Do not let any failure block or fail the actual API response to the user.
	if h.auditClient != nil {
		actorID := "SYSTEM"
		actorType := "SYSTEM"
		if authCtx := auth.GetAuthContext(r.Context()); authCtx != nil {
			if authCtx.User != nil {
				actorID = authCtx.User.UserID
				actorType = "USER"
			} else if authCtx.Client != nil {
				actorID = authCtx.Client.ClientID
				actorType = "SYSTEM"
			}
		}

		var status string
		var msg string
		if err != nil {
			status = "FAILURE"
			msg = fmt.Sprintf("Failed to read task info: %v", err)
		} else {
			status = "SUCCESS"
			msg = fmt.Sprintf("User read task info for task %s", taskId)
		}

		metadata := map[string]any{
			"task_id": taskId,
		}

		auditLog := audit.AuditLogRequest{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			EventType:  "SYSTEM_EVENT",
			Action:     "READ_TASK_INFO",
			Status:     status,
			ActorID:    actorID,
			ActorType:  actorType,
			TargetID:   &taskId,
			TargetType: "TASK",
			Message:    []byte(msg),
			Metadata:   metadata,
		}

		h.auditClient.LogEvent(context.WithoutCancel(r.Context()), &auditLog)
	}

	if err != nil {
		// Differentiate between invalid ID/NotFound and internal errors, if necessary.
		status := http.StatusInternalServerError
		if strings.HasPrefix(err.Error(), "taskID is invalid") {
			status = http.StatusBadRequest
		} else if strings.HasPrefix(err.Error(), "task ") {
			status = http.StatusNotFound
		}
		writeJSONError(w, status, err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, result)
}

// HandleExecuteTask is an HTTP handler for executing a task via POST request
func (h *HTTPHandler) HandleExecuteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ExecuteTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	result, err := h.manager.ExecuteTask(r.Context(), req)

	// Fire the audit log asynchronously before returning the HTTP response.
	// Do not let any failure block or fail the actual API response to the user.
	if h.auditClient != nil {
		actorID := "SYSTEM"
		actorType := "SYSTEM"
		if authCtx := auth.GetAuthContext(r.Context()); authCtx != nil {
			if authCtx.User != nil {
				actorID = authCtx.User.UserID
				actorType = "USER"
			} else if authCtx.Client != nil {
				actorID = authCtx.Client.ClientID
				actorType = "SYSTEM"
			}
		}

		var status string
		var msg string
		if err != nil {
			status = "FAILURE"
			msg = fmt.Sprintf("Task execution failed: %v", err)
		} else {
			status = "SUCCESS"
			msg = fmt.Sprintf("User executed task %s", req.TaskID)
		}

		metadata := map[string]any{
			"workflow_id": req.WorkflowID,
		}
		if req.Payload != nil {
			if payloadBytes, err := json.Marshal(req.Payload); err == nil {
				var payloadMap map[string]any
				if err := json.Unmarshal(payloadBytes, &payloadMap); err == nil {
					metadata["payload"] = payloadMap
				} else {
					metadata["payload_raw"] = string(payloadBytes)
				}
			}
		}

		auditLog := audit.AuditLogRequest{
			Timestamp:  time.Now().UTC().Format(time.RFC3339),
			EventType:  "SYSTEM_EVENT",
			Action:     "EXECUTE_TASK",
			Status:     status,
			ActorID:    actorID,
			ActorType:  actorType,
			TargetID:   &req.TaskID,
			TargetType: "TASK",
			Message:    []byte(msg),
			Metadata:   metadata,
		}

		h.auditClient.LogEvent(context.WithoutCancel(r.Context()), &auditLog)
	}

	if err != nil {
		status := http.StatusInternalServerError
		if string(err.Error()) == "task_id is required" {
			status = http.StatusBadRequest
		} else if len(err.Error()) >= 5 && string(err.Error()[:5]) == "task " {
			status = http.StatusNotFound
		}
		writeJSONError(w, status, err.Error())
		return
	}

	// Return success response
	writeJSONResponse(w, http.StatusOK, result.ApiResponse)
}

func writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSONResponse(w, status, ExecuteTaskResponse{
		Success: false,
		Error:   message,
	})
}
