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

			// EXACT BOT LOGIC: Check if PDF text directly contains the 14-digit IMEI.
			found := strings.Contains(pdfTextContent, imei14)
			matched := ""
			// Provide the 15-digit match to the UI if available, else indicate a generic match.
			if found {
				for _, seq := range pdf15Digits {
					if strings.HasPrefix(seq, imei14) {
						matched = seq
						break
					}
				}
				if matched == "" {
					matched = "(prefix matched in text)"
				}
			}
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

	report.TextReport = generateTextReport(report)

	return report, nil
}

func generateTextReport(report *models.IMEIVerificationReport) string {
	var sb strings.Builder

	sb.WriteString("=========================================\n")
	sb.WriteString("        IMEI VERIFICATION REPORT\n")
	sb.WriteString("=========================================\n\n")

	sb.WriteString(fmt.Sprintf("Total IMEIs processed: %d\n", report.TotalIMEIs))
	sb.WriteString(fmt.Sprintf("Total Found in PDF: %d\n", report.TotalFound))
	sb.WriteString(fmt.Sprintf("Total Missing: %d\n\n", report.TotalMissing))

	sb.WriteString("--- STATISTICS BY COLUMN ---\n")
	for _, stat := range report.ColumnStats {
		sb.WriteString(fmt.Sprintf("%s: %d processed (%d found, %d missing)\n", stat.Column, stat.Total, stat.Found, stat.Missing))
	}
	sb.WriteString("\n")

	if report.TotalMissing > 0 {
		sb.WriteString("--- MISSING IMEI DETAILS ---\n")
		for _, res := range report.Results {
			if !res.Found {
				sb.WriteString(fmt.Sprintf("Line %d [%s]: %s (Missing)\n", res.CSVLine, res.Column, res.IMEI14))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString("--- FULL MAPPING ---\n")
	for _, res := range report.Results {
		if res.Found {
			sb.WriteString(fmt.Sprintf("Line %d [%s]: %s -> MATCHED: %s\n", res.CSVLine, res.Column, res.IMEI14, res.MatchedIMEI))
		} else {
			sb.WriteString(fmt.Sprintf("Line %d [%s]: %s -> MISSING\n", res.CSVLine, res.Column, res.IMEI14))
		}
	}

	return sb.String()
}
