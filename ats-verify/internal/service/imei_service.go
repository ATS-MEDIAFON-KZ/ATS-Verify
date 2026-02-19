package service

import (
	"encoding/csv"
	"fmt"
	"io"
	"regexp"
	"strings"

	"ats-verify/internal/models"
)

// IMEIService handles IMEI verification logic.
type IMEIService struct{}

// NewIMEIService creates a new IMEIService.
func NewIMEIService() *IMEIService {
	return &IMEIService{}
}

// imeiColumns lists the CSV column names to scan for IMEI values.
var imeiColumns = []string{"imei", "imei1", "imei2", "imei3", "imei4", "imei_number"}

// regex15Digits matches 15-digit sequences in PDF text for IMEI extraction.
var regex15Digits = regexp.MustCompile(`\b\d{15}\b`)

// Analyze compares IMEIs from a multi-column CSV against text extracted from a PDF.
// CSV columns: Imei1..Imei4 (any subset). PDF text: 15-digit sequences.
// Match rule: 14-digit IMEI (from CSV) must be a prefix of a 15-digit sequence (from PDF).
func (s *IMEIService) Analyze(csvReader io.Reader, pdfTextContent string) (*models.IMEIVerificationReport, error) {
	reader := csv.NewReader(csvReader)
	reader.TrimLeadingSpace = true

	// Read header and find IMEI columns.
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}

	// Map: column index → column name (only IMEI columns).
	colMap := make(map[int]string)
	for i, col := range header {
		lower := strings.ToLower(strings.TrimSpace(col))
		for _, target := range imeiColumns {
			if lower == target {
				colMap[i] = strings.TrimSpace(col)
				break
			}
		}
	}
	if len(colMap) == 0 {
		return nil, fmt.Errorf("CSV must contain at least one IMEI column (imei, imei1..imei4)")
	}

	// Extract all 15-digit sequences from PDF.
	pdf15Digits := regex15Digits.FindAllString(pdfTextContent, -1)

	// Per-column stats tracker.
	statsMap := make(map[string]*models.IMEIColumnStats)
	for _, colName := range colMap {
		statsMap[colName] = &models.IMEIColumnStats{Column: colName}
	}

	report := &models.IMEIVerificationReport{}
	csvLine := 1 // header is line 1, data starts at 2

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		csvLine++

		for colIdx, colName := range colMap {
			if colIdx >= len(record) {
				continue
			}

			imei := strings.TrimSpace(record[colIdx])
			if imei == "" {
				continue
			}

			// Normalize: take first 14 digits if longer.
			imei14 := imei
			if len(imei14) > 14 {
				imei14 = imei14[:14]
			}
			if len(imei14) < 14 {
				continue // Skip invalid short IMEIs.
			}

			report.TotalIMEIs++
			statsMap[colName].Total++

			// Search: does any 15-digit PDF sequence start with this 14-digit prefix?
			matched := ""
			for _, seq := range pdf15Digits {
				if strings.HasPrefix(seq, imei14) {
					matched = seq
					break
				}
			}

			found := matched != ""
			if found {
				report.TotalFound++
				statsMap[colName].Found++
			} else {
				report.TotalMissing++
				statsMap[colName].Missing++
			}

			report.Results = append(report.Results, models.IMEIMatchResult{
				CSVLine:     csvLine,
				Column:      colName,
				IMEI14:      imei14,
				MatchedIMEI: matched,
				Found:       found,
			})
		}
	}

	// Collect per-column stats in deterministic order.
	for _, colName := range colMap {
		report.ColumnStats = append(report.ColumnStats, *statsMap[colName])
	}

	return report, nil
}
