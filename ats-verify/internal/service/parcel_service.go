package service

import (
	"context"
	"encoding/csv"
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
// Expected CSV format: marketplace, country, brand, product_name, track_number, snt, date
func (s *ParcelService) ProcessCSVUpload(ctx context.Context, reader io.Reader, overrideMarketplace string, uploadedBy uuid.UUID) (*UploadResult, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true

	// Read header row
	header, err := csvReader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}

	// Build column index map
	colMap := make(map[string]int)
	for i, col := range header {
		colMap[strings.ToLower(strings.TrimSpace(col))] = i
	}

	// Require track_number column
	trackIdx, ok := colMap["track_number"]
	if !ok {
		// Try alternative names
		for _, alt := range []string{"track", "tracking_number", "трек-номер", "трек_номер"} {
			if idx, found := colMap[alt]; found {
				trackIdx = idx
				ok = true
				break
			}
		}
		if !ok {
			return nil, fmt.Errorf("CSV must contain a 'track_number' column")
		}
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

		result.TotalProcessed++

		trackNumber := strings.TrimSpace(record[trackIdx])
		if trackNumber == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: empty track_number", result.TotalProcessed))
			continue
		}

		country := getCSVField(record, colMap, "country")
		brand := getCSVField(record, colMap, "brand")
		productName := getCSVField(record, colMap, "product_name")
		if productName == "" { // fallback alternative column naming
			productName = getCSVField(record, colMap, "name")
		}
		snt := getCSVField(record, colMap, "snt")
		dateStr := getCSVField(record, colMap, "date")

		rowMarketplace := getCSVField(record, colMap, "marketplace")

		// Validation of required fields per GOALS.md
		if overrideMarketplace == "" && rowMarketplace == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: missing marketplace column for admin upload", result.TotalProcessed))
			continue
		}
		if country == "" || brand == "" || productName == "" || snt == "" || dateStr == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("row %d: missing required fields (country, brand, name, snt, date)", result.TotalProcessed))
			continue
		}

		finalMarketplace := overrideMarketplace
		if finalMarketplace == "" {
			finalMarketplace = rowMarketplace
		}

		var uploadDate time.Time
		uploadDate, err = time.Parse("2006-01-02", dateStr) // assuming YYYY-MM-DD
		if err != nil {
			uploadDate = time.Now() // fallback to now if unparseable, though strict validation could block it
		}

		parcel := &models.Parcel{
			TrackNumber: trackNumber,
			Marketplace: finalMarketplace,
			Country:     country,
			Brand:       brand,
			ProductName: productName,
			SNT:         snt,
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
