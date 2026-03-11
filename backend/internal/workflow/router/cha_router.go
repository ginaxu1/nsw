package router

import (
	"encoding/json"
	"net/http"

	"github.com/OpenNSW/nsw/internal/workflow/service"
)

type CHARouter struct {
	chaService *service.CHAService
}

func NewCHARouter(chaService *service.CHAService) *CHARouter {
	return &CHARouter{chaService: chaService}
}

// HandleGetCHAs handles GET /api/v1/chas
func (cr *CHARouter) HandleGetCHAs(w http.ResponseWriter, r *http.Request) {
	chas, err := cr.chaService.ListCHAs(r.Context())
	if err != nil {
		http.Error(w, "failed to retrieve CHAs: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(chas); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
