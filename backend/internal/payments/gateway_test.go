package payments

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestLankaPayAdapter_CreateIntent(t *testing.T) {
	adapter := NewLankaPayAdapter()

	ctx := context.Background()
	amount := decimal.NewFromFloat(150.0)
	currency := "LKR"
	ref := "LANKA-123"
	meta := map[string]string{"task_id": "SRC-456"}

	resp, err := adapter.CreateIntent(ctx, amount, currency, ref, meta)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	if resp.GatewaySessionID == "" {
		t.Errorf("expected session id to be generated")
	}

	if resp.CheckoutURL == "" {
		t.Errorf("expected checkout URL to be generated")
	}

	if time.Until(resp.ExpiresAt) < 23*time.Hour {
		t.Errorf("expected expiry to be roughly 24 hours from now, got %v", resp.ExpiresAt)
	}
}
