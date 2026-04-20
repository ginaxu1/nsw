package payments

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/OpenNSW/nsw/internal/events"
	"github.com/google/uuid"
)

// PaymentService defines the business logic operations for Payments.
type PaymentService interface {
	CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error)
	ValidateReference(ctx context.Context, req ValidateReferenceRequest) (*ValidateReferenceResponse, error)
	ProcessWebhook(ctx context.Context, payload WebhookPayload) error
}

type paymentService struct {
	repo       PaymentRepository
	dispatcher events.EventDispatcher
	gateway    PaymentGateway
}

// NewPaymentService initializes a new payment service.
func NewPaymentService(repo PaymentRepository, dispatcher events.EventDispatcher, gateway PaymentGateway) PaymentService {
	return &paymentService{repo: repo, dispatcher: dispatcher, gateway: gateway}
}

// CreateCheckoutSession saves the initial intent and returns gateway session details.
func (s *paymentService) CreateCheckoutSession(ctx context.Context, req CreateCheckoutRequest) (*CreateCheckoutResponse, error) {
	sourceID, ok := req.Metadata["task_id"]
	if !ok {
		return nil, fmt.Errorf("task_id is required in metadata")
	}
	gatewayResp, err := s.gateway.CreateIntent(ctx, req.Amount, req.Currency, req.ReferenceNumber, req.Metadata)
	if err != nil {
		return nil, fmt.Errorf("gateway failed to create intent: %w", err)
	}

	tx := &PaymentTransaction{
		ID:              uuid.NewString(),
		ReferenceNumber: req.ReferenceNumber,
		TaskID:          sourceID,
		SessionID:       gatewayResp.GatewaySessionID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		Status:          PaymentStatusPending,
		ExpiryDate:      gatewayResp.ExpiresAt,
		GatewayMetadata: req.Metadata,
	}

	if err := s.repo.Create(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to create payment transaction: %w", err)
	}

	slog.Info("created checkout session", "reference_number", req.ReferenceNumber, "session_id", gatewayResp.GatewaySessionID)

	return &CreateCheckoutResponse{
		SessionID:   gatewayResp.GatewaySessionID,
		CheckoutURL: gatewayResp.CheckoutURL,
		ExpiresIn:   int(time.Until(gatewayResp.ExpiresAt).Seconds()),
	}, nil
}

// ValidateReference is called by GovPay when a user searches for their reference number.
func (s *paymentService) ValidateReference(ctx context.Context, req ValidateReferenceRequest) (*ValidateReferenceResponse, error) {
	slog.Info("validating incoming payment reference", "reference", req.PaymentReference)

	tx, err := s.repo.GetByReferenceNumber(ctx, req.PaymentReference)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment reference: %w", err)
	}
	if tx == nil {
		return &ValidateReferenceResponse{IsPayable: false, Remarks: "Invalid reference number"}, nil
	}

	isPayable := tx.Status == PaymentStatusPending && time.Now().Before(tx.ExpiryDate)

	return &ValidateReferenceResponse{
		Amount:     tx.Amount,
		Currency:   tx.Currency,
		TraderName: "Sample Trader", // TODO: Fetch from actual domain models/context
		OGAName:    "Sample OGA",    // TODO: Fetch from actual domain models/context
		ExpiryDate: tx.ExpiryDate.Format(time.RFC3339),
		IsPayable:  isPayable,
		Remarks:    fmt.Sprintf("Current status: %s", tx.Status),
	}, nil
}

// ProcessWebhook processes asynchronous success/failure updates from GovPay.
func (s *paymentService) ProcessWebhook(ctx context.Context, payload WebhookPayload) error {
	slog.Info("processing payment webhook", "reference_number", payload.ReferenceNumber, "status", payload.Status)

	tx, err := s.repo.GetByReferenceNumber(ctx, payload.ReferenceNumber)
	if err != nil {
		return fmt.Errorf("failed to retrieve payment by reference: %w", err)
	}
	if tx == nil {
		return fmt.Errorf("payment reference not found: %s", payload.ReferenceNumber)
	}

	// Idempotency: Ignore if we already recorded a final status
	if tx.Status == payload.Status || tx.Status == PaymentStatusSuccess {
		slog.Info("webhook ignored (idempotent)", "reference", tx.ReferenceNumber, "current_status", tx.Status)
		return nil
	}

	tx.Status = payload.Status
	tx.PaymentMethod = payload.PaymentMethod

	if tx.GatewayMetadata == nil {
		tx.GatewayMetadata = make(map[string]string)
	}
	tx.GatewayMetadata["gateway_transaction_id"] = payload.GatewayTransactionID
	tx.GatewayMetadata["webhook_timestamp"] = payload.Timestamp

	if err := s.repo.Update(ctx, tx); err != nil {
		return fmt.Errorf("failed to update payment transaction status: %w", err)
	}

	slog.Info("payment transaction updated successfully", "reference", tx.ReferenceNumber, "status", tx.Status)

	// Phase 1: Event-Driven Notification
	// Ensure loose coupling by publishing to the internal event dispatcher
	// rather than calling task manager or workflow functions synchronously.
	if tx.Status == PaymentStatusSuccess && s.dispatcher != nil {
		s.dispatcher.Publish(ctx, events.Event{
			Type: EventPaymentCompleted,
			Payload: InternalPaymentEvent{
				EventType: EventPaymentCompleted,
				Data: EventData{
					TaskID:               tx.TaskID,
					ReferenceNumber:      tx.ReferenceNumber,
					GatewayTransactionID: tx.GatewayMetadata["gateway_transaction_id"],
					Status:               tx.Status,
					AmountPaid:           tx.Amount,
					Currency:             tx.Currency,
					ConfirmedAt:          payload.Timestamp,
				},
			},
		})
	}

	return nil
}
