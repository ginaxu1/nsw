package persistence

import (
	"context"
	"fmt"

	"github.com/OpenNSW/nsw/internal/task/plugin"
	"gorm.io/gorm"
)

type paymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository creates a new instance of plugin.PaymentRepository.
func NewPaymentRepository(db *gorm.DB) plugin.PaymentRepository {
	return &paymentRepository{db: db}
}

func (r *paymentRepository) CreateTransaction(ctx context.Context, trx *plugin.PaymentTransactionDB) error {
	return r.db.WithContext(ctx).Create(trx).Error
}

func (r *paymentRepository) GetTransactionByReference(ctx context.Context, ref string) (*plugin.PaymentTransactionDB, error) {
	var trx plugin.PaymentTransactionDB
	err := r.db.WithContext(ctx).Where("reference_number = ?", ref).First(&trx).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("transaction not found for reference: %s", ref)
		}
		return nil, err
	}
	return &trx, nil
}

func (r *paymentRepository) GetTransactionByExecutionID(ctx context.Context, execID string) (*plugin.PaymentTransactionDB, error) {
	var trx plugin.PaymentTransactionDB
	err := r.db.WithContext(ctx).Where("execution_id = ?", execID).First(&trx).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("transaction not found for execution ID: %s", execID)
		}
		return nil, err
	}
	return &trx, nil
}

func (r *paymentRepository) UpdateTransactionStatus(ctx context.Context, ref string, status string) error {
	return r.db.WithContext(ctx).Model(&plugin.PaymentTransactionDB{}).
		Where("reference_number = ?", ref).
		Update("status", status).Error
}
