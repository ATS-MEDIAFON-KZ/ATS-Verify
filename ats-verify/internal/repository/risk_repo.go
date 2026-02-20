package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"ats-verify/internal/models"
)

// RiskRepository handles IIN/BIN risk profile operations.
type RiskRepository struct {
	db *sql.DB
}

// NewRiskRepository creates a new RiskRepository.
func NewRiskRepository(db *sql.DB) *RiskRepository {
	return &RiskRepository{db: db}
}

// Upsert creates or updates a risk profile for an IIN/BIN.
func (r *RiskRepository) Upsert(ctx context.Context, profile *models.IINBINRisk) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO iin_bin_risks (id, iin_bin, risk_level, flagged_by, comment, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		 ON CONFLICT (iin_bin)
		 DO UPDATE SET risk_level = EXCLUDED.risk_level, flagged_by = EXCLUDED.flagged_by, comment = EXCLUDED.comment, updated_at = NOW()`,
		uuid.New(), profile.IINBIN, profile.RiskLevel, profile.FlaggedBy, profile.Comment,
	)
	if err != nil {
		return fmt.Errorf("upserting risk profile: %w", err)
	}
	return nil
}

// GetByIINBIN retrieves a risk profile by IIN/BIN.
func (r *RiskRepository) GetByIINBIN(ctx context.Context, iinBin string) (*models.IINBINRisk, error) {
	var p models.IINBINRisk
	err := r.db.QueryRowContext(ctx,
		`SELECT id, iin_bin, risk_level, flagged_by, comment, created_at, updated_at
		 FROM iin_bin_risks WHERE iin_bin = $1`,
		iinBin,
	).Scan(&p.ID, &p.IINBIN, &p.RiskLevel, &p.FlaggedBy, &p.Comment, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying risk profile: %w", err)
	}
	return &p, nil
}

// ListAll returns all risk profiles.
func (r *RiskRepository) ListAll(ctx context.Context) ([]models.IINBINRisk, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT id, iin_bin, risk_level, flagged_by, comment, created_at, updated_at FROM iin_bin_risks ORDER BY updated_at DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("listing risk profiles: %w", err)
	}
	defer rows.Close()

	var profiles []models.IINBINRisk
	for rows.Next() {
		var p models.IINBINRisk
		if err := rows.Scan(&p.ID, &p.IINBIN, &p.RiskLevel, &p.FlaggedBy, &p.Comment, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning risk profile: %w", err)
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}

// Delete removes a risk profile by ID.
func (r *RiskRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM iin_bin_risks WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("deleting risk profile: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("risk profile not found")
	}
	return nil
}
