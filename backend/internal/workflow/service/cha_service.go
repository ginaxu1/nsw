package service

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/OpenNSW/nsw/internal/workflow/model"
)

type CHAService struct {
	db *gorm.DB
}

func NewCHAService(db *gorm.DB) *CHAService {
	return &CHAService{db: db}
}

// ListCHAs returns all customs house agents ordered by name.
func (s *CHAService) ListCHAs(ctx context.Context) ([]model.CHA, error) {
	var chas []model.CHA
	if err := s.db.WithContext(ctx).Order("name ASC").Find(&chas).Error; err != nil {
		return nil, fmt.Errorf("failed to retrieve customs house agents: %w", err)
	}
	return chas, nil
}
