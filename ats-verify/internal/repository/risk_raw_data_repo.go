package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"ats-verify/internal/models"
)

// RiskRawDataRepository handles interactions with the risk_raw_data table used for analytics.
type RiskRawDataRepository struct {
	db *sql.DB
}

// NewRiskRawDataRepository creates a new RiskRawDataRepository.
func NewRiskRawDataRepository(db *sql.DB) *RiskRawDataRepository {
	return &RiskRawDataRepository{db: db}
}

// BulkInsert inserts multiple raw risk data rows into the database efficiently.
func (r *RiskRawDataRepository) BulkInsert(ctx context.Context, rows []models.RiskRawData) error {
	if len(rows) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	// PostgreSQL supports batch insert via multi-value VALUES clause
	query := `INSERT INTO risk_raw_data (
		report_date, application_id, iin_bin, document, user_name, organization, status, reject, reason, created_at
	) VALUES `

	valStrings := make([]string, 0, len(rows))
	valArgs := make([]interface{}, 0, len(rows)*10)
	i := 1

	for _, row := range rows {
		valStrings = append(valStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, NOW())", i, i+1, i+2, i+3, i+4, i+5, i+6, i+7, i+8))
		valArgs = append(valArgs, row.ReportDate, row.ApplicationID, row.IINBIN, row.Document, row.UserName, row.Organization, row.Status, row.Reject, row.Reason)
		i += 9
	}

	query += strings.Join(valStrings, ",")

	_, err = tx.ExecContext(ctx, query, valArgs...)
	if err != nil {
		return fmt.Errorf("bulk insert query: %w", err)
	}

	return tx.Commit()
}

// DocumentReuseFlag indicates the same document used across different IINs/BINs.
type DocumentReuseFlag struct {
	DocNumber string `json:"doc_number" db:"document"`
	Count     int    `json:"count" db:"count"`
}

// GetDocumentReuseReport Returns documents used more than once.
func (r *RiskRawDataRepository) GetDocumentReuseReport(ctx context.Context) ([]DocumentReuseFlag, error) {
	query := `
		SELECT document, COUNT(*) as count 
		FROM risk_raw_data 
		WHERE document IS NOT NULL AND document != ''
		GROUP BY document 
		HAVING COUNT(*) > 1 
		ORDER BY count DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetDocumentReuseReport query: %w", err)
	}
	defer rows.Close()

	var results []DocumentReuseFlag
	for rows.Next() {
		var flag DocumentReuseFlag
		if err := rows.Scan(&flag.DocNumber, &flag.Count); err != nil {
			return nil, err
		}
		results = append(results, flag)
	}
	return results, nil
}

// GetDocumentIINReuseReport Returns documents used by MORE THAN ONE DISTINCT IIN/BIN.
func (r *RiskRawDataRepository) GetDocumentIINReuseReport(ctx context.Context) ([]DocumentReuseFlag, error) {
	query := `
		SELECT document, COUNT(DISTINCT iin_bin) as count 
		FROM risk_raw_data 
		WHERE document IS NOT NULL AND document != ''
		GROUP BY document 
		HAVING COUNT(DISTINCT iin_bin) > 1 
		ORDER BY count DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetDocumentIINReuseReport query: %w", err)
	}
	defer rows.Close()

	var results []DocumentReuseFlag
	for rows.Next() {
		var flag DocumentReuseFlag
		if err := rows.Scan(&flag.DocNumber, &flag.Count); err != nil {
			return nil, err
		}
		results = append(results, flag)
	}
	return results, nil
}

// FrequencyFlag indicates an IIN/BIN with unusually high application count.
type FrequencyFlag struct {
	IINBIN string `json:"iin_bin" db:"iin_bin"`
	Count  int    `json:"count" db:"count"`
}

// GetIINFrequencyReport Returns IINs grouped by frequency, sorted desc.
func (r *RiskRawDataRepository) GetIINFrequencyReport(ctx context.Context) ([]FrequencyFlag, error) {
	query := `
		SELECT iin_bin, COUNT(*) as count 
		FROM risk_raw_data 
		GROUP BY iin_bin 
		ORDER BY count DESC
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetIINFrequencyReport query: %w", err)
	}
	defer rows.Close()

	var results []FrequencyFlag
	for rows.Next() {
		var flag FrequencyFlag
		if err := rows.Scan(&flag.IINBIN, &flag.Count); err != nil {
			return nil, err
		}
		results = append(results, flag)
	}
	return results, nil
}

// FlipFlopFlag indicates a document with contradictory status changes over time.
type FlipFlopFlag struct {
	DocNumber string `json:"doc_number"`
	IINBIN    string `json:"iin_bin"`
	Statuses  string `json:"statuses"` // aggregated view "status A -> status B"
}

// GetFlipFlopStatusReport Detects flip-flop statuses for the same document over time.
func (r *RiskRawDataRepository) GetFlipFlopStatusReport(ctx context.Context) ([]FlipFlopFlag, error) {
	// A simple approach is finding documents that have > 1 distinct status.
	// For deeper time-based flip-flop, string_agg can group statuses ordered by date.
	query := `
		SELECT document, iin_bin, string_agg(status, ' -> ' ORDER BY report_date ASC) as statuses
		FROM risk_raw_data
		WHERE document IS NOT NULL AND document != ''
		GROUP BY document, iin_bin
		HAVING COUNT(DISTINCT status) > 1
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetFlipFlopStatusReport query: %w", err)
	}
	defer rows.Close()

	var results []FlipFlopFlag
	for rows.Next() {
		var flag FlipFlopFlag
		if err := rows.Scan(&flag.DocNumber, &flag.IINBIN, &flag.Statuses); err != nil {
			return nil, err
		}
		results = append(results, flag)
	}
	return results, nil
}
