package service

import (
	"context"
	"fmt"
	"time"

	"ats-verify/internal/models"
)

// Tracker defines a unified interface for external parcel tracking providers.
// Clean Architecture: handler/service depend on this interface, not concrete clients.
type Tracker interface {
	// Track retrieves tracking events for a given track number.
	// Returns nil slice (not error) if the provider does not recognize the track number.
	Track(ctx context.Context, trackNumber string) ([]models.TrackingEvent, error)

	// Provider returns the name of the tracking provider (e.g. "CDEK", "Kazpost").
	Provider() string
}

// TrackingService aggregates multiple Tracker implementations and queries them in order.
type TrackingService struct {
	trackers []Tracker
}

// NewTrackingService creates a TrackingService with the given tracker implementations.
func NewTrackingService(trackers ...Tracker) *TrackingService {
	return &TrackingService{trackers: trackers}
}

// TrackingResult holds the combined result from all providers.
type TrackingResult struct {
	TrackNumber string                 `json:"track_number"`
	Events      []models.TrackingEvent `json:"events"`
	Provider    string                 `json:"provider"`
}

// Track queries each provider in order and returns the first successful result.
func (s *TrackingService) Track(ctx context.Context, trackNumber string) (*TrackingResult, error) {
	for _, t := range s.trackers {
		events, err := t.Track(ctx, trackNumber)
		if err != nil {
			continue // Try next provider on error.
		}
		if len(events) > 0 {
			return &TrackingResult{
				TrackNumber: trackNumber,
				Events:      events,
				Provider:    t.Provider(),
			}, nil
		}
	}
	return nil, fmt.Errorf("no tracking data found for %s", trackNumber)
}

// --- CDEK Client Stub ---

// CDEKTracker implements the Tracker interface for CDEK API v2.
// TODO (Phase 3.4): Replace stub with real HTTP client using OAuth2 bearer token.
type CDEKTracker struct {
	// baseURL   string
	// authToken string
}

// NewCDEKTracker creates a new CDEKTracker.
func NewCDEKTracker() *CDEKTracker {
	return &CDEKTracker{}
}

func (c *CDEKTracker) Provider() string { return "CDEK" }

func (c *CDEKTracker) Track(ctx context.Context, trackNumber string) ([]models.TrackingEvent, error) {
	// Stub: return mock events for any track starting with "CDEK" or containing digits.
	// Real implementation: GET https://api.cdek.ru/v2/orders?cdek_number={trackNumber}
	return []models.TrackingEvent{
		{
			StatusCode:  "ACCEPTED",
			Description: "Посылка принята на склад СДЭК",
			Location:    "Москва, Россия",
			EventTime:   time.Now().Add(-72 * time.Hour),
			Source:      "CDEK",
		},
		{
			StatusCode:  "IN_TRANSIT",
			Description: "В пути",
			Location:    "Москва → Алматы",
			EventTime:   time.Now().Add(-48 * time.Hour),
			Source:      "CDEK",
		},
	}, nil
}

// --- Kazpost Client Stub ---

// KazpostTracker implements the Tracker interface for Kazpost API.
// TODO (Phase 3.4): Replace stub with real HTTP client.
type KazpostTracker struct {
	// baseURL string
}

// NewKazpostTracker creates a new KazpostTracker.
func NewKazpostTracker() *KazpostTracker {
	return &KazpostTracker{}
}

func (k *KazpostTracker) Provider() string { return "Kazpost" }

func (k *KazpostTracker) Track(ctx context.Context, trackNumber string) ([]models.TrackingEvent, error) {
	// Stub: return mock events.
	// Real implementation: POST https://open.post.kz/...
	return []models.TrackingEvent{
		{
			StatusCode:  "RECEIVED",
			Description: "Почтовое отправление принято",
			Location:    "Алматы, Казахстан",
			EventTime:   time.Now().Add(-24 * time.Hour),
			Source:      "Kazpost",
		},
	}, nil
}
