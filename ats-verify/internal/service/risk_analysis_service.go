package service

import (
	"context"
	"fmt"
	"io"
	"log"
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
	DocumentReuse    []repository.DocumentReuseFlag    `json:"document_reuse"`
	DocumentIINReuse []repository.DocumentIINReuseFlag `json:"document_iin_reuse"`
	IINFrequency     []repository.FrequencyFlag        `json:"high_frequency_iin"`
	FlipFlopStatus   []repository.FlipFlopFlag         `json:"flip_flop"`
}

// AnalyzeCSV processes the risk analysis CSV and detects anomalies.
// Expected CSV strictly index-based:
// 0: Date
// 1: AppId
// 2: IIN/BIN
// 3: doc
// 4: User
// 5: Org
// 6: Status
// 7: Reject
// 8: Reason
func (s *RiskAnalysisService) AnalyzeCSV(ctx context.Context, reader io.Reader, flaggedBy uuid.UUID) (int, error) {
	csvReader, err := NewRobustCSVReader(reader)
	if err != nil {
		return 0, fmt.Errorf("initializing robust CSV reader: %w", err)
	}

	// Ignore header
	_, err = csvReader.Read()
	if err != nil {
		return 0, fmt.Errorf("reading CSV header: %w", err)
	}

	var rows []RiskCSVRow
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		for i := range record {
			record[i] = strings.TrimSpace(record[i])
			if record[i] == "<nil>" {
				record[i] = ""
			}
		}

		if len(record) < 9 {
			log.Printf("Risk CSV: skipping row due to length < 9 (len=%d) record: %v", len(record), record)
			continue // Skip incomplete rows to avoid panic
		}

		row := RiskCSVRow{
			Date:   record[0],
			AppID:  record[1],
			IINBIN: record[2],
			DocNum: record[3],
			User:   record[4],
			Org:    record[5],
			Status: record[6],
			Reject: record[7],
			Reason: record[8],
		}

		if row.IINBIN == "" || row.IINBIN == "0" {
			log.Printf("Risk CSV: skipping row due to empty IIN/BIN. docNum=%s", row.DocNum)
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
