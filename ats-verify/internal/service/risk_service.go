package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"ats-verify/internal/models"
	"ats-verify/internal/repository"
)

// RiskService handles risk profile business logic.
type RiskService struct {
	riskRepo *repository.RiskRepository
}

// NewRiskService creates a new RiskService.
func NewRiskService(riskRepo *repository.RiskRepository) *RiskService {
	return &RiskService{riskRepo: riskRepo}
}

// List returns all risk profiles.
func (s *RiskService) List(ctx context.Context) ([]models.IINBINRisk, error) {
	return s.riskRepo.ListAll(ctx)
}

// CreateOrUpdate creates or updates a risk profile.
func (s *RiskService) CreateOrUpdate(ctx context.Context, iinBin string, riskLevel models.RiskLevel, comment string, flaggedBy uuid.UUID) error {
	// Validate risk level
	switch riskLevel {
	case models.RiskGreen, models.RiskYellow, models.RiskRed:
	default:
		return fmt.Errorf("invalid risk level: %s", riskLevel)
	}

	if iinBin == "" {
		return fmt.Errorf("iin_bin is required")
	}

	profile := &models.IINBINRisk{
		IINBIN:    iinBin,
		RiskLevel: riskLevel,
		FlaggedBy: flaggedBy,
		Comment:   comment,
	}

	return s.riskRepo.Upsert(ctx, profile)
}

// Delete removes a risk profile by ID.
func (s *RiskService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.riskRepo.Delete(ctx, id)
}
