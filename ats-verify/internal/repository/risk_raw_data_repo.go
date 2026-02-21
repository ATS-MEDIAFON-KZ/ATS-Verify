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

// BulkInsert inserts multiple raw risk data rows into the database efficiently in chunks.
func (r *RiskRawDataRepository) BulkInsert(ctx context.Context, rows []models.RiskRawData) error {
	if len(rows) == 0 {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	chunkSize := 5000 // 5000 rows * 9 parameters = 45000 params, well under 65535 Postgre limit.
	for start := 0; start < len(rows); start += chunkSize {
		end := start + chunkSize
		if end > len(rows) {
			end = len(rows)
		}

		batch := rows[start:end]

		query := `INSERT INTO risk_raw_data (
			report_date, application_id, iin_bin, document, user_name, organization, status, reject, reason, created_at
		) VALUES `

		valStrings := make([]string, 0, len(batch))
		valArgs := make([]interface{}, 0, len(batch)*9)
		i := 1

		for _, row := range batch {
			valStrings = append(valStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, NOW())", i, i+1, i+2, i+3, i+4, i+5, i+6, i+7, i+8))
			valArgs = append(valArgs, row.ReportDate, row.ApplicationID, row.IINBIN, row.Document, row.UserName, row.Organization, row.Status, row.Reject, row.Reason)
			i += 9
		}

		query += strings.Join(valStrings, ",")

		_, err = tx.ExecContext(ctx, query, valArgs...)
		if err != nil {
			return fmt.Errorf("bulk insert query at chunk %d: %w", start, err)
		}
	}

	return tx.Commit()
}

// DocumentReuseFlag indicates the same document used across different IINs/BINs.
type DocumentReuseFlag struct {
	DocNumber  string `json:"document_number"`
	UsageCount int    `json:"usage_count"`
	LastUsed   string `json:"last_used"`
}

// GetDocumentReuseReport Returns documents used more than once.
func (r *RiskRawDataRepository) GetDocumentReuseReport(ctx context.Context) ([]DocumentReuseFlag, error) {
	query := `
		SELECT document, COUNT(*) as usage_count, COALESCE(MAX(report_date::text), MAX(created_at::text)) as last_used
		FROM risk_raw_data 
		WHERE document IS NOT NULL AND document != ''
		GROUP BY document 
		HAVING COUNT(*) > 1 
		ORDER BY usage_count DESC
		LIMIT 100
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetDocumentReuseReport query: %w", err)
	}
	defer rows.Close()

	var results []DocumentReuseFlag
	for rows.Next() {
		var flag DocumentReuseFlag
		if err := rows.Scan(&flag.DocNumber, &flag.UsageCount, &flag.LastUsed); err != nil {
			return nil, err
		}
		results = append(results, flag)
	}
	return results, nil
}

type DocumentIINReuseFlag struct {
	DocNumber string `json:"document_number"`
	IINCount  int    `json:"iin_count"`
	IINs      string `json:"iins"`
}

// GetDocumentIINReuseReport Returns documents used by MORE THAN ONE DISTINCT IIN/BIN.
func (r *RiskRawDataRepository) GetDocumentIINReuseReport(ctx context.Context) ([]DocumentIINReuseFlag, error) {
	query := `
		SELECT document, COUNT(DISTINCT iin_bin) as iin_count, string_agg(DISTINCT iin_bin, ', ') as iins
		FROM risk_raw_data 
		WHERE document IS NOT NULL AND document != ''
		GROUP BY document 
		HAVING COUNT(DISTINCT iin_bin) > 1 
		ORDER BY iin_count DESC
		LIMIT 100
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetDocumentIINReuseReport query: %w", err)
	}
	defer rows.Close()

	var results []DocumentIINReuseFlag
	for rows.Next() {
		var flag DocumentIINReuseFlag
		if err := rows.Scan(&flag.DocNumber, &flag.IINCount, &flag.IINs); err != nil {
			return nil, err
		}
		results = append(results, flag)
	}
	return results, nil
}

// FrequencyFlag indicates an IIN/BIN with unusually high application count.
type FrequencyFlag struct {
	IINBIN     string `json:"iin"`
	UsageCount int    `json:"usage_count"`
	LastUsed   string `json:"last_used"`
}

// GetIINFrequencyReport Returns IINs grouped by frequency, sorted desc.
func (r *RiskRawDataRepository) GetIINFrequencyReport(ctx context.Context) ([]FrequencyFlag, error) {
	query := `
		SELECT iin_bin, COUNT(*) as usage_count, COALESCE(MAX(report_date::text), MAX(created_at::text)) as last_used
		FROM risk_raw_data 
		WHERE iin_bin IS NOT NULL AND iin_bin != '' AND iin_bin != '0'
		GROUP BY iin_bin 
		HAVING COUNT(*) > 5
		ORDER BY usage_count DESC
		LIMIT 100
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetIINFrequencyReport query: %w", err)
	}
	defer rows.Close()

	var results []FrequencyFlag
	for rows.Next() {
		var flag FrequencyFlag
		if err := rows.Scan(&flag.IINBIN, &flag.UsageCount, &flag.LastUsed); err != nil {
			return nil, err
		}
		results = append(results, flag)
	}
	return results, nil
}

// FlipFlopFlag indicates a document with contradictory status changes over time.
type FlipFlopFlag struct {
	DocNumber     string `json:"document_number"`
	ApprovedCount int    `json:"approved_count"`
	RejectedCount int    `json:"rejected_count"`
}

// GetFlipFlopStatusReport Detects flip-flop statuses for the same document over time.
func (r *RiskRawDataRepository) GetFlipFlopStatusReport(ctx context.Context) ([]FlipFlopFlag, error) {
	query := `
		SELECT document, 
               SUM(CASE WHEN status ILIKE '%одобрен%' OR status ILIKE '%принят%' OR status ILIKE '%выдан%' OR status ILIKE '%утвержден%' THEN 1 ELSE 0 END) as approved_count,
               SUM(CASE WHEN status ILIKE '%отказ%' OR status ILIKE '%отклонен%' THEN 1 ELSE 0 END) as rejected_count
		FROM risk_raw_data
		WHERE document IS NOT NULL AND document != ''
		GROUP BY document
		HAVING SUM(CASE WHEN status ILIKE '%одобрен%' OR status ILIKE '%принят%' OR status ILIKE '%выдан%' OR status ILIKE '%утвержден%' THEN 1 ELSE 0 END) > 0 
           AND SUM(CASE WHEN status ILIKE '%отказ%' OR status ILIKE '%отклонен%' THEN 1 ELSE 0 END) > 0
        ORDER BY 
            (SUM(CASE WHEN status ILIKE '%одобрен%' OR status ILIKE '%принят%' OR status ILIKE '%выдан%' OR status ILIKE '%утвержден%' THEN 1 ELSE 0 END) + 
             SUM(CASE WHEN status ILIKE '%отказ%' OR status ILIKE '%отклонен%' THEN 1 ELSE 0 END)) DESC
		LIMIT 100
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("GetFlipFlopStatusReport query: %w", err)
	}
	defer rows.Close()

	var results []FlipFlopFlag
	for rows.Next() {
		var flag FlipFlopFlag
		if err := rows.Scan(&flag.DocNumber, &flag.ApprovedCount, &flag.RejectedCount); err != nil {
			return nil, err
		}
		results = append(results, flag)
	}
	return results, nil
}
