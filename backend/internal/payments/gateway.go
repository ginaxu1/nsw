package payments

import (
	"context"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// PaymentGateway defines the strategy for interacting with external payment providers.
type PaymentGateway interface {
	CreateIntent(ctx context.Context, amount decimal.Decimal, currency string, reference string, metadata map[string]string) (*GatewayIntentResponse, error)
}

// GatewayIntentResponse contains the details returned by a payment gateway when initiating a session.
type GatewayIntentResponse struct {
	GatewaySessionID string
	CheckoutURL      string
	ExpiresAt        time.Time
}

// LankaPayAdapter implements PaymentGateway for the LankaPay sandbox/gateway.
type LankaPayAdapter struct {
	CheckoutBaseURL string
	DefaultExpiry   time.Duration
}

// NewLankaPayAdapter creates a new LankaPay adapter.
func NewLankaPayAdapter() *LankaPayAdapter {
	return &LankaPayAdapter{
		CheckoutBaseURL: "https://sandbox.govpay.lk/checkout/",
		DefaultExpiry:   24 * time.Hour,
	}
}

// CreateIntent generates a mocked checkout URL for LankaPay.
func (a *LankaPayAdapter) CreateIntent(_ context.Context, _ decimal.Decimal, _ string, _ string, _ map[string]string) (*GatewayIntentResponse, error) {
	// In a real implementation, this would make an outbound HTTP call to LankaPay's API.
	// For now, we replicate the existing mock behavior.
	sessionID := "sess_" + fmt.Sprintf("%d", time.Now().UnixNano())
	checkoutURL := a.CheckoutBaseURL + sessionID

	return &GatewayIntentResponse{
		GatewaySessionID: sessionID,
		CheckoutURL:      checkoutURL,
		ExpiresAt:        time.Now().Add(a.DefaultExpiry),
	}, nil
}
