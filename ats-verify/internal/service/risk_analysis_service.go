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
	riskRepo *repository.RiskRepository
}

// NewRiskAnalysisService creates a new RiskAnalysisService.
func NewRiskAnalysisService(riskRepo *repository.RiskRepository) *RiskAnalysisService {
	return &RiskAnalysisService{riskRepo: riskRepo}
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

// RiskAnalysisResult is the output of the bulk risk analysis.
type RiskAnalysisResult struct {
	TotalRows        int                 `json:"total_rows"`
	UniqueIINs       int                 `json:"unique_iins"`
	DocumentReuse    []DocumentReuseFlag `json:"document_reuse"`
	HighFrequencyIIN []FrequencyFlag     `json:"high_frequency_iin"`
	FlipFlopStatus   []FlipFlopFlag      `json:"flip_flop_status"`
	AutoFlagged      int                 `json:"auto_flagged"`
}

// DocumentReuseFlag indicates the same document used across different IINs.
type DocumentReuseFlag struct {
	DocNumber string   `json:"doc_number"`
	IINs      []string `json:"iins"`
	Count     int      `json:"count"`
}

// FrequencyFlag indicates an IIN/BIN with unusually high application count.
type FrequencyFlag struct {
	IINBIN    string `json:"iin_bin"`
	Count     int    `json:"count"`
	RiskLevel string `json:"risk_level"`
}

// FlipFlopFlag indicates an IIN/BIN with contradictory status changes.
type FlipFlopFlag struct {
	IINBIN   string   `json:"iin_bin"`
	Statuses []string `json:"statuses"`
	AppIDs   []string `json:"app_ids"`
}

// AnalyzeCSV processes the risk analysis CSV and detects anomalies.
// CSV format: Date | AppId | IIN/BIN | doc | User | Org | Status | Reject | Reason
func (s *RiskAnalysisService) AnalyzeCSV(ctx context.Context, reader io.Reader, flaggedBy uuid.UUID) (*RiskAnalysisResult, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
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
			return nil, fmt.Errorf("CSV missing required column: %s (tried: %v)", req, altNames[req])
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
			continue
		}
		rows = append(rows, row)
	}

	if len(rows) == 0 {
		return nil, fmt.Errorf("CSV contains no valid data rows")
	}

	result := &RiskAnalysisResult{TotalRows: len(rows)}

	// --- Analysis 1: Document Reuse ---
	// Same doc number used by different IINs → suspicious.
	docToIINs := make(map[string]map[string]bool)
	for _, r := range rows {
		if r.DocNum == "" {
			continue
		}
		if docToIINs[r.DocNum] == nil {
			docToIINs[r.DocNum] = make(map[string]bool)
		}
		docToIINs[r.DocNum][r.IINBIN] = true
	}
	for doc, iins := range docToIINs {
		if len(iins) > 1 {
			iinList := make([]string, 0, len(iins))
			for iin := range iins {
				iinList = append(iinList, iin)
			}
			result.DocumentReuse = append(result.DocumentReuse, DocumentReuseFlag{
				DocNumber: doc,
				IINs:      iinList,
				Count:     len(iinList),
			})
		}
	}

	// --- Analysis 2: High-Frequency IINs ---
	// IIN/BIN appearing > threshold times → flag.
	iinFreq := make(map[string]int)
	uniqueIINs := make(map[string]bool)
	for _, r := range rows {
		iinFreq[r.IINBIN]++
		uniqueIINs[r.IINBIN] = true
	}
	result.UniqueIINs = len(uniqueIINs)

	const yellowThreshold = 5
	const redThreshold = 10
	for iin, count := range iinFreq {
		if count >= yellowThreshold {
			level := "yellow"
			if count >= redThreshold {
				level = "red"
			}
			result.HighFrequencyIIN = append(result.HighFrequencyIIN, FrequencyFlag{
				IINBIN:    iin,
				Count:     count,
				RiskLevel: level,
			})
		}
	}

	// --- Analysis 3: Flip-Flop Status ---
	// Same IIN with contradictory statuses (e.g. approved then rejected).
	iinStatuses := make(map[string][]string)
	iinAppIDs := make(map[string][]string)
	for _, r := range rows {
		iinStatuses[r.IINBIN] = append(iinStatuses[r.IINBIN], r.Status)
		if r.AppID != "" {
			iinAppIDs[r.IINBIN] = append(iinAppIDs[r.IINBIN], r.AppID)
		}
	}
	for iin, statuses := range iinStatuses {
		uniqueStatuses := uniqueStrings(statuses)
		if len(uniqueStatuses) > 1 {
			result.FlipFlopStatus = append(result.FlipFlopStatus, FlipFlopFlag{
				IINBIN:   iin,
				Statuses: uniqueStatuses,
				AppIDs:   iinAppIDs[iin],
			})
		}
	}

	// --- Auto-flag to DB ---
	// Auto-assign risk levels for high-frequency IINs.
	for _, hf := range result.HighFrequencyIIN {
		riskLevel := models.RiskYellow
		if hf.RiskLevel == "red" {
			riskLevel = models.RiskRed
		}
		err := s.riskRepo.Upsert(ctx, &models.RiskProfile{
			IINBIN:    hf.IINBIN,
			RiskLevel: riskLevel,
			FlaggedBy: flaggedBy,
			Reason:    fmt.Sprintf("Auto-flagged: %d applications detected", hf.Count),
		})
		if err == nil {
			result.AutoFlagged++
		}
	}

	// Auto-flag document reuse IINs as yellow.
	for _, dr := range result.DocumentReuse {
		for _, iin := range dr.IINs {
			err := s.riskRepo.Upsert(ctx, &models.RiskProfile{
				IINBIN:    iin,
				RiskLevel: models.RiskYellow,
				FlaggedBy: flaggedBy,
				Reason:    fmt.Sprintf("Auto-flagged: document %s reused across %d IINs", dr.DocNumber, dr.Count),
			})
			if err == nil {
				result.AutoFlagged++
			}
		}
	}

	return result, nil
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
