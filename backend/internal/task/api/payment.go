package api

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/OpenNSW/nsw/internal/config"
	taskManager "github.com/OpenNSW/nsw/internal/task/manager"
	"github.com/OpenNSW/nsw/internal/task/persistence"
	"github.com/OpenNSW/nsw/internal/task/plugin"
	"gorm.io/gorm"
)

type PaymentHandler struct {
	tm   taskManager.TaskManager
	cfg  *config.Config
	db   *gorm.DB
	repo plugin.PaymentRepository
}

func NewPaymentHandler(tm taskManager.TaskManager, cfg *config.Config, db *gorm.DB) *PaymentHandler {
	return &PaymentHandler{
		tm:   tm,
		cfg:  cfg,
		db:   db,
		repo: persistence.NewPaymentRepository(db),
	}
}

type PaymentCallbackRequest struct {
	ReferenceNumber string `json:"reference_number"`
	Status          string `json:"status"` // "SUCCESS" or "FAILED"
	// Signature     string `json:"signature"`
}

func (h *PaymentHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := r.PathValue("provider")
	slog.Info("received payment callback", "provider", provider)

	// Abstract validation and parsing based on the provider
	switch provider {
	case "govpay":
		// TODO(GovPay): Implement actual checksum/signature verification
		if !h.cfg.Payment.MockMode {
			signature := r.Header.Get("GovPay-Signature")
			if signature == "" {
				h.writeError(w, http.StatusUnauthorized, "Missing signature")
				return
			}
			// if !verifySignature(req, signature, h.cfg.Payment.Secret) { ... }
		}
	default:
		// For now, allow generic handling or return error for unknown providers
		// For a clean agnostic approach, we should fail for unknown providers
		http.Error(w, fmt.Sprintf("Unsupported payment provider: %s", provider), http.StatusBadRequest)
		return
	}

	req, ok := h.decodeCallbackRequest(w, r)
	if !ok {
		return
	}

	h.processPayment(w, r, req)
}

func (h *PaymentHandler) HandleMockCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !h.cfg.Payment.MockMode {
		h.writeError(w, http.StatusForbidden, "Mock mode disabled")
		return
	}

	req, ok := h.decodeCallbackRequest(w, r)
	if !ok {
		return
	}

	h.processPayment(w, r, req)
}

func (h *PaymentHandler) processPayment(w http.ResponseWriter, r *http.Request, req PaymentCallbackRequest) {
	ctx := r.Context()
	slog.InfoContext(ctx, "received Payment callback", "reference", req.ReferenceNumber, "status", req.Status)

	// Validate status explicitly to avoid incorrect success processing
	var action string
	switch req.Status {
	case "SUCCESS":
		action = plugin.PaymentActionSuccess
	case "FAILED":
		action = plugin.PaymentActionFailed
	default:
		slog.ErrorContext(ctx, "received unexpected payment status", "status", req.Status, "reference", req.ReferenceNumber)
		h.writeError(w, http.StatusBadRequest, "Invalid payment status received")
		return
	}

	// Use a transaction to ensure atomicity between DB update and TaskManager execution
	err := h.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		transaction, err := h.repo.GetTransactionByReference(ctx, req.ReferenceNumber)
		if err != nil {
			return err
		}

		if transaction.Status != "PENDING" {
			slog.InfoContext(ctx, "transaction already processed", "reference", req.ReferenceNumber, "status", transaction.Status)
			return nil // Already processed, return success to gateway
		}

		// 1. Update DB status
		dbStatus := "COMPLETED"
		if req.Status == "FAILED" {
			dbStatus = "FAILED"
		}
		if err := tx.Model(&transaction).Update("status", dbStatus).Error; err != nil {
			return err
		}

		// 2. Execute Task event
		execReq := taskManager.ExecuteTaskRequest{
			TaskID: transaction.TaskID,
			Payload: &plugin.ExecutionRequest{
				Action: action,
			},
		}

		_, err = h.tm.ExecuteTask(ctx, execReq)
		if err != nil {
			return fmt.Errorf("failed to transition task: %w", err)
		}

		return nil
	})

	if err != nil {
		slog.ErrorContext(ctx, "failed to process payment", "error", err)
		// Mask internal errors from public endpoint
		h.writeError(w, http.StatusInternalServerError, "An internal error occurred while processing the payment")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]bool{"success": true})
}

func (h *PaymentHandler) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		slog.Error("failed to encode JSON response", "error", err)
	}
}

func (h *PaymentHandler) writeError(w http.ResponseWriter, status int, message string) {
	h.writeJSON(w, status, map[string]interface{}{
		"success": false,
		"error":   message,
	})
}

func (h *PaymentHandler) decodeCallbackRequest(w http.ResponseWriter, r *http.Request) (PaymentCallbackRequest, bool) {
	var req PaymentCallbackRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.writeError(w, http.StatusBadRequest, "Invalid request body")
		return PaymentCallbackRequest{}, false
	}
	return req, true
}

func (h *PaymentHandler) HandleTransactionInquiry(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	provider := r.PathValue("provider")

	// Basic shared secret check
	apiKey := r.Header.Get("X-API-Key")
	if subtle.ConstantTimeCompare([]byte(apiKey), []byte(h.cfg.Payment.InquiryAPIKey)) != 1 {
		h.writeError(w, http.StatusUnauthorized, "Invalid API Key")
		return
	}

	reference := r.PathValue("reference")
	if reference == "" {
		h.writeError(w, http.StatusBadRequest, "Missing reference parameter")
		return
	}

	trx, err := h.repo.GetTransactionByReference(r.Context(), reference)
	if err != nil {
		slog.ErrorContext(r.Context(), "inquiry failed", "ref", reference, "error", err)
		h.writeError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	// Dynamic response based on provider
	var currency string
	switch provider {
	case "govpay":
		currency = "LKR"
	default:
		currency = "UNK"
	}

	resp := map[string]interface{}{
		"provider":         provider,
		"reference_number": trx.ReferenceNumber,
		"amount":           trx.Amount,
		"currency":         currency,
		"status":           trx.Status,
	}

	h.writeJSON(w, http.StatusOK, resp)
}
