package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"

	"ats-verify/internal/models"
	"ats-verify/internal/repository"
)

// ParcelService handles parcel business logic.
type ParcelService struct {
	parcelRepo *repository.ParcelRepository
}

// NewParcelService creates a new ParcelService.
func NewParcelService(parcelRepo *repository.ParcelRepository) *ParcelService {
	return &ParcelService{parcelRepo: parcelRepo}
}

// UploadResult summarizes a CSV upload operation.
type UploadResult struct {
	TotalProcessed int      `json:"total_processed"`
	Inserted       int      `json:"inserted"`
	Updated        int      `json:"updated"`
	Skipped        int      `json:"skipped"`
	Errors         []string `json:"errors"`
}

// ProcessCSVUpload parses a CSV file and upserts parcels.
// Expected CSV format strictly index-based:
// 0: marketplace
// 1: country
// 2: brand
// 3: product_name
// 4: track_number
// 5: snt
// 6: date
func (s *ParcelService) ProcessCSVUpload(ctx context.Context, reader io.Reader, overrideMarketplace string, uploadedBy uuid.UUID) (*UploadResult, error) {
	csvReader, err := NewRobustCSVReader(reader)
	if err != nil {
		return nil, fmt.Errorf("initializing robust CSV reader: %w", err)
	}

	// Read header row to map columns dynamically
	headerRow, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}

	colMap := make(map[string]int)
	for i, colName := range headerRow {
		cleaned := strings.ToLower(strings.TrimSpace(colName))
		// Remove BOM or surrounding quotes just in case
		cleaned = strings.Trim(cleaned, "\xef\xbb\xbf\"'")
		colMap[cleaned] = i
	}

	result := &UploadResult{}

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("row parse error: %v", err))
			continue
		}

		// Clean up the record
		for i := range record {
			record[i] = strings.TrimSpace(record[i])
			if record[i] == "<nil>" {
				record[i] = ""
			}
		}

		result.TotalProcessed++

		rowMarketplace := getCSVField(record, colMap, "marketplace")
		country := getCSVField(record, colMap, "country")
		if country == "" {
			country = getCSVField(record, colMap, "страна")
		}

		brand := getCSVField(record, colMap, "brand")
		if brand == "" {
			brand = getCSVField(record, colMap, "бренд")
		}

		productName := getCSVField(record, colMap, "name")
		if productName == "" {
			productName = getCSVField(record, colMap, "product_name")
		}
		if productName == "" {
			productName = getCSVField(record, colMap, "товар")
		}
		if productName == "" {
			productName = getCSVField(record, colMap, "название")
		}

		trackNumber := getCSVField(record, colMap, "track_number")
		if trackNumber == "" {
			trackNumber = getCSVField(record, colMap, "трек-номер")
		}
		if trackNumber == "" {
			trackNumber = getCSVField(record, colMap, "track")
		}

		snt := getCSVField(record, colMap, "snt")
		if snt == "" {
			snt = getCSVField(record, colMap, "снт")
		}

		dateStr := getCSVField(record, colMap, "date")
		if dateStr == "" {
			dateStr = getCSVField(record, colMap, "дата")
		}

		isUsed := false
		usedStr := strings.ToUpper(getCSVField(record, colMap, "used"))
		if usedStr == "" {
			usedStr = strings.ToUpper(getCSVField(record, colMap, "is_used"))
		}
		if usedStr == "" {
			usedStr = strings.ToUpper(getCSVField(record, colMap, "использован"))
		}

		if usedStr == "TRUE" || usedStr == "1" || usedStr == "YES" || usedStr == "ДА" {
			isUsed = true
		}

		if trackNumber == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: empty track_number", result.TotalProcessed))
			continue
		}

		finalMarketplace := overrideMarketplace
		if finalMarketplace == "" {
			finalMarketplace = rowMarketplace
		}

		// Strictly skip rows where ANY critical value is missing
		var missing []string
		if finalMarketplace == "" {
			missing = append(missing, "marketplace")
		}
		if country == "" {
			missing = append(missing, "country")
		}
		if brand == "" {
			missing = append(missing, "brand")
		}
		if productName == "" {
			missing = append(missing, "name")
		}

		if len(missing) > 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: missing required fields: %s", result.TotalProcessed, strings.Join(missing, ", ")))
			continue
		}

		var uploadDate time.Time
		if dateStr != "" {
			parsedDate, err := time.Parse("2006-01-02", dateStr)
			if err == nil {
				uploadDate = parsedDate
			} else {
				parsedDate2, err2 := time.Parse("02.01.2006", dateStr) // standard KZ format fallback
				if err2 == nil {
					uploadDate = parsedDate2
				} else {
					uploadDate = time.Now()
				}
			}
		}

		parcel := &models.Parcel{
			TrackNumber: trackNumber,
			Marketplace: finalMarketplace,
			Country:     country,
			Brand:       brand,
			ProductName: productName,
			SNT:         snt,
			IsUsed:      isUsed,
			UploadDate:  uploadDate,
			UploadedBy:  uploadedBy,
		}

		upsertResult, err := s.parcelRepo.UpsertParcel(ctx, parcel)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d (%s): %v", result.TotalProcessed, trackNumber, err))
			continue
		}

		switch upsertResult.Action {
		case "inserted":
			result.Inserted++
		case "updated":
			result.Updated++
		case "skipped_used":
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", trackNumber, upsertResult.Message))
		}
	}

	return result, nil
}

// ProcessJSONUpload parses JSON payloads (array of parcels) and upserts.
type JSONUploadRequest struct {
	Marketplace string `json:"marketplace"`
	Country     string `json:"country"`
	Brand       string `json:"brand"`
	ProductName string `json:"name"`
	TrackNumber string `json:"track_number"`
	SNT         string `json:"snt"`
	Date        string `json:"date"`
	Used        bool   `json:"used,omitempty"`
}

func (s *ParcelService) ProcessJSONUpload(ctx context.Context, payloads []JSONUploadRequest, overrideMarketplace string, uploadedBy uuid.UUID) (*UploadResult, error) {
	result := &UploadResult{}

	for i, req := range payloads {
		result.TotalProcessed++

		trackNumber := strings.TrimSpace(req.TrackNumber)
		if trackNumber == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("item %d: empty track_number", i))
			continue
		}

		finalMarketplace := overrideMarketplace
		if finalMarketplace == "" {
			finalMarketplace = strings.TrimSpace(req.Marketplace)
		}

		country := strings.TrimSpace(req.Country)
		brand := strings.TrimSpace(req.Brand)
		productName := strings.TrimSpace(req.ProductName)
		snt := strings.TrimSpace(req.SNT)
		dateStr := strings.TrimSpace(req.Date)

		// Strictly skip incomplete rows
		var missing []string
		if finalMarketplace == "" {
			missing = append(missing, "marketplace")
		}
		if country == "" {
			missing = append(missing, "country")
		}
		if brand == "" {
			missing = append(missing, "brand")
		}
		if productName == "" {
			missing = append(missing, "name")
		}
		if snt == "" {
			missing = append(missing, "snt")
		}
		if dateStr == "" {
			missing = append(missing, "date")
		}

		if len(missing) > 0 {
			result.Errors = append(result.Errors, fmt.Sprintf("item %d: missing required fields: %s", i, strings.Join(missing, ", ")))
			continue
		}

		var uploadDate time.Time
		uploadDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			uploadDate = time.Now()
		}

		parcel := &models.Parcel{
			TrackNumber: trackNumber,
			Marketplace: finalMarketplace,
			Country:     country,
			Brand:       brand,
			ProductName: productName,
			SNT:         snt,
			IsUsed:      req.Used,
			UploadDate:  uploadDate,
			UploadedBy:  uploadedBy,
		}

		upsertResult, err := s.parcelRepo.UpsertParcel(ctx, parcel)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("item %d (%s): %v", i, trackNumber, err))
			continue
		}

		switch upsertResult.Action {
		case "inserted":
			result.Inserted++
		case "updated":
			result.Updated++
			result.Errors = append(result.Errors, fmt.Sprintf("Warning: %s %s", trackNumber, upsertResult.Message))
		case "skipped_used":
			result.Skipped++
			result.Errors = append(result.Errors, fmt.Sprintf("Warning: %s: %s", trackNumber, upsertResult.Message))
		}
	}

	return result, nil
}

// ListParcelsFilter holds filter parameters for listing parcels.
type ListParcelsFilter struct {
	Status string // "used", "unused", "" (all)
	Search string
	Page   int
	Limit  int
}

// ListParcelsResponse is the paginated parcels list.
type ListParcelsResponse struct {
	Parcels []models.Parcel `json:"parcels"`
	Total   int             `json:"total"`
	Page    int             `json:"page"`
	Limit   int             `json:"limit"`
}

// ListParcels returns filtered and paginated parcels.
func (s *ParcelService) ListParcels(ctx context.Context, f ListParcelsFilter) (*ListParcelsResponse, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.Limit < 1 || f.Limit > 100 {
		f.Limit = 20
	}

	parcels, total, err := s.parcelRepo.ListWithFilters(ctx, f.Status, f.Search, f.Page, f.Limit)
	if err != nil {
		return nil, fmt.Errorf("listing parcels: %w", err)
	}

	return &ListParcelsResponse{
		Parcels: parcels,
		Total:   total,
		Page:    f.Page,
		Limit:   f.Limit,
	}, nil
}

// MarkParcelUsed sets the is_used flag to true in the database.
func (s *ParcelService) MarkParcelUsed(ctx context.Context, trackNumber string) error {
	return s.parcelRepo.MarkUsed(ctx, trackNumber)
}

// BulkTrackResult is the result of looking up multiple track numbers.
type BulkTrackResult struct {
	TrackNumber string         `json:"track_number"`
	Found       bool           `json:"found"`
	Parcel      *models.Parcel `json:"parcel,omitempty"`
}

// BulkTrackLookup looks up multiple track numbers at once.
func (s *ParcelService) BulkTrackLookup(ctx context.Context, tracks []string) ([]BulkTrackResult, error) {
	results := make([]BulkTrackResult, 0, len(tracks))

	foundParcels, err := s.parcelRepo.BulkLookup(ctx, tracks)
	if err != nil {
		return nil, fmt.Errorf("bulk lookup: %w", err)
	}

	// Build map for quick lookup
	foundMap := make(map[string]*models.Parcel)
	for i := range foundParcels {
		foundMap[foundParcels[i].TrackNumber] = &foundParcels[i]
	}

	for _, track := range tracks {
		track = strings.TrimSpace(track)
		if track == "" {
			continue
		}
		if p, ok := foundMap[track]; ok {
			results = append(results, BulkTrackResult{
				TrackNumber: track,
				Found:       true,
				Parcel:      p,
			})
		} else {
			results = append(results, BulkTrackResult{
				TrackNumber: track,
				Found:       false,
			})
		}
	}

	return results, nil
}

func getCSVField(record []string, colMap map[string]int, field string) string {
	if idx, ok := colMap[field]; ok && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}
