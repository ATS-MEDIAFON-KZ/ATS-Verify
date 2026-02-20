package service

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/google/uuid"

	"ats-verify/internal/models"
	"ats-verify/internal/repository"
)

// RiskAnalysisService handles advanced risk analysis logic.
// Analyzes CSV uploads to detect document reuse, flip-flop statuses,
// and high-frequency IIN/BINs per GOALS.md specification.
type RiskAnalysisService struct {
	riskRepo    *repository.RiskRepository
	riskRawRepo *repository.RiskRawDataRepository
}

// NewRiskAnalysisService creates a new RiskAnalysisService.
func NewRiskAnalysisService(riskRepo *repository.RiskRepository, riskRawRepo *repository.RiskRawDataRepository) *RiskAnalysisService {
	return &RiskAnalysisService{riskRepo: riskRepo, riskRawRepo: riskRawRepo}
}

// RiskCSVRow represents a parsed row from the risk analysis CSV.
// Format: Date | AppId | IIN/BIN | doc | User | Org | Status | Reject | Reason
type RiskCSVRow struct {
	Date   string
	AppID  string
	IINBIN string
	DocNum string
	User   string
	Org    string
	Status string
	Reject string
	Reason string
}

// Reports output structures
type AnalyticsReports struct {
	DocumentReuse    []repository.DocumentReuseFlag `json:"document_reuse"`
	DocumentIINReuse []repository.DocumentReuseFlag `json:"document_iin_reuse"`
	IINFrequency     []repository.FrequencyFlag     `json:"iin_frequency"`
	FlipFlopStatus   []repository.FlipFlopFlag      `json:"flip_flop_status"`
}

// AnalyzeCSV processes the risk analysis CSV and detects anomalies.
// CSV format: Date | AppId | IIN/BIN | doc | User | Org | Status | Reject | Reason
func (s *RiskAnalysisService) AnalyzeCSV(ctx context.Context, reader io.Reader, flaggedBy uuid.UUID) (int, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	header, err := csvReader.Read()
	if err != nil {
		return 0, fmt.Errorf("reading CSV header: %w", err)
	}

	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Verify required columns exist.
	requiredCols := []string{"iin", "doc", "status"}
	altNames := map[string][]string{
		"iin":    {"iin/bin", "iin_bin", "iin", "bin"},
		"doc":    {"doc", "document", "doc_number", "document_number"},
		"status": {"status"},
	}

	resolvedCols := make(map[string]int)
	for _, req := range requiredCols {
		found := false
		for _, alt := range altNames[req] {
			if idx, ok := colMap[alt]; ok {
				resolvedCols[req] = idx
				found = true
				break
			}
		}
		if !found {
			return 0, fmt.Errorf("CSV missing required column: %s (tried: %v)", req, altNames[req])
		}
	}

	// Optional columns.
	appIDIdx := -1
	if idx, ok := colMap["appid"]; ok {
		appIDIdx = idx
	} else if idx, ok := colMap["app_id"]; ok {
		appIDIdx = idx
	}

	// Parse all rows.
	var rows []RiskCSVRow
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		row := RiskCSVRow{
			IINBIN: safeGet(record, resolvedCols["iin"]),
			DocNum: safeGet(record, resolvedCols["doc"]),
			Status: safeGet(record, resolvedCols["status"]),
		}
		if appIDIdx >= 0 {
			row.AppID = safeGet(record, appIDIdx)
		}

		if row.IINBIN == "" {
			continue // IIN is strictly required for any risk logic
		}
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return 0, fmt.Errorf("CSV contains no valid data rows")
	}

	// Dump into DB
	var dbRows []models.RiskRawData
	for _, r := range rows {
		dbRows = append(dbRows, models.RiskRawData{
			// report date omitted or parsed specifically if available.
			ApplicationID: r.AppID,
			IINBIN:        r.IINBIN,
			Document:      r.DocNum,
			UserName:      r.User,
			Organization:  r.Org,
			Status:        r.Status,
			Reject:        r.Reject,
			Reason:        r.Reason,
		})
	}

	if err := s.riskRawRepo.BulkInsert(ctx, dbRows); err != nil {
		return 0, fmt.Errorf("bulk inserting risk data: %w", err)
	}

	// We let the Analytics endpoints handle fetching reports.
	// The DB will do the heavy lifting from now on.

	return len(dbRows), nil
}

// GetAnalyticsReports fetches all 4 risk analytics combinations.
func (s *RiskAnalysisService) GetAnalyticsReports(ctx context.Context) (*AnalyticsReports, error) {
	docReuse, _ := s.riskRawRepo.GetDocumentReuseReport(ctx)
	docIinReuse, _ := s.riskRawRepo.GetDocumentIINReuseReport(ctx)
	iinFreq, _ := s.riskRawRepo.GetIINFrequencyReport(ctx)
	flipFlop, _ := s.riskRawRepo.GetFlipFlopStatusReport(ctx)

	return &AnalyticsReports{
		DocumentReuse:    docReuse,
		DocumentIINReuse: docIinReuse,
		IINFrequency:     iinFreq,
		FlipFlopStatus:   flipFlop,
	}, nil
}

func safeGet(record []string, idx int) string {
	if idx >= 0 && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}

func uniqueStrings(ss []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
