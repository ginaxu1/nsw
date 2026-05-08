package manager

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHTTPHandler_HandleExecuteTask(t *testing.T) {
	t.Run("Invalid Method", func(t *testing.T) {
		tm := &taskManager{}
		handler := NewHTTPHandler(tm, nil)
		req := httptest.NewRequest(http.MethodGet, "/execute", nil)
		w := httptest.NewRecorder()

		handler.HandleExecuteTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
	})

	t.Run("Invalid Body", func(t *testing.T) {
		tm := &taskManager{}
		handler := NewHTTPHandler(tm, nil)
		req := httptest.NewRequest(http.MethodPost, "/execute", bytes.NewBufferString("invalid json"))
		w := httptest.NewRecorder()

		handler.HandleExecuteTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestHTTPHandler_HandleGetTask(t *testing.T) {
	t.Run("Missing TaskID", func(t *testing.T) {
		tm, _, _, _ := setupTest(t)
		handler := NewHTTPHandler(tm, nil)
		req := httptest.NewRequest(http.MethodGet, "/tasks/", nil)
		// No path value set
		w := httptest.NewRecorder()

		handler.HandleGetTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("Invalid TaskID string", func(t *testing.T) {
		tm, _, mockStore, _ := setupTest(t)
		handler := NewHTTPHandler(tm, nil)
		req := httptest.NewRequest(http.MethodGet, "/tasks/invalid", nil)
		req.SetPathValue("id", "invalid")
		w := httptest.NewRecorder()

		mockStore.On("GetByID", "invalid").Return(nil, errors.New("not found")).Once()

		handler.HandleGetTask(w, req)

		resp := w.Result()
		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
