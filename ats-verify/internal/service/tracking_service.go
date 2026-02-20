package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"ats-verify/internal/models"

	"github.com/google/uuid"
)

// Tracker defines a unified interface for external parcel tracking providers.
type Tracker interface {
	Track(ctx context.Context, trackNumber string) ([]models.TrackingEvent, error)
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
	ExternalURL string                 `json:"external_url,omitempty"`
}

var numericOnly = regexp.MustCompile(`^\d+$`)

// Track queries each provider in order and returns the first successful result.
// If CDEK's API blocks the request, it returns a redirect link to the CDEK tracking page.
func (s *TrackingService) Track(ctx context.Context, trackNumber string) (*TrackingResult, error) {
	for _, t := range s.trackers {
		events, err := t.Track(ctx, trackNumber)
		if err != nil {
			continue
		}
		if len(events) > 0 {
			return &TrackingResult{
				TrackNumber: trackNumber,
				Events:      events,
				Provider:    t.Provider(),
			}, nil
		}
	}

	// If no provider returned data and the track looks like a CDEK number (pure digits),
	// return a redirect to CDEK's own tracking page (their API blocks server-side requests).
	if numericOnly.MatchString(trackNumber) {
		return &TrackingResult{
			TrackNumber: trackNumber,
			Events:      nil,
			Provider:    "CDEK",
			ExternalURL: fmt.Sprintf("https://www.cdek.ru/ru/tracking/?order_id=%s", trackNumber),
		}, nil
	}

	return nil, fmt.Errorf("no tracking data found for %s", trackNumber)
}

// ─── Kazpost Real Client ────────────────────────────────────────────────

// kazpostEventsResponse matches the JSON from post.kz/external-api/tracking/api/v2/{id}/events
type kazpostEventsResponse struct {
	Events []kazpostDayGroup `json:"events"`
}

type kazpostDayGroup struct {
	Date     string            `json:"date"` // "05.09.2025"
	Activity []kazpostActivity `json:"activity"`
}

type kazpostActivity struct {
	Time   string   `json:"time"` // "14:35:56"
	Zip    string   `json:"zip"`
	City   string   `json:"city"`
	Name   string   `json:"name"`
	Status []string `json:"status"`
}

// KazpostTracker implements the Tracker interface for Kazpost public tracking API.
type KazpostTracker struct {
	client  *http.Client
	baseURL string
}

// NewKazpostTracker creates a new KazpostTracker.
func NewKazpostTracker() *KazpostTracker {
	return &KazpostTracker{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL: "https://post.kz/external-api/tracking/api/v2",
	}
}

func (k *KazpostTracker) Provider() string { return "Kazpost" }

func (k *KazpostTracker) Track(ctx context.Context, trackNumber string) ([]models.TrackingEvent, error) {
	url := fmt.Sprintf("%s/%s/events", k.baseURL, trackNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("kazpost: creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://post.kz/services/postal/"+trackNumber)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36")

	resp, err := k.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("kazpost: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kazpost: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("kazpost: reading body: %w", err)
	}

	var data kazpostEventsResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("kazpost: parsing json: %w", err)
	}

	if len(data.Events) == 0 {
		return nil, nil
	}

	var events []models.TrackingEvent
	for _, dayGroup := range data.Events {
		for _, act := range dayGroup.Activity {
			eventTime, _ := time.Parse("02.01.2006 15:04:05", dayGroup.Date+" "+act.Time)

			statusDesc := translateKazpostStatus(act.Status)
			location := act.City
			if location == "" {
				location = act.Name
			}

			events = append(events, models.TrackingEvent{
				ID:          uuid.New(),
				StatusCode:  strings.Join(act.Status, ","),
				Description: statusDesc,
				Location:    location,
				EventTime:   eventTime,
				Source:      "Kazpost",
			})
		}
	}

	return events, nil
}

// translateKazpostStatus converts Kazpost status codes to human-readable Russian descriptions.
func translateKazpostStatus(statuses []string) string {
	if len(statuses) == 0 {
		return "Неизвестный статус"
	}

	translations := map[string]string{
		"Registered":       "Принято почтовое отправление",
		"EMA":              "Отправлено",
		"SRT_CUSTOM":       "Прибыло в центр сортировки",
		"DetainedByCustom": "Задержано таможней",
		"Checking":         "Проверка",
		"PaymentRequired":  "Требуется оплата",
		"CORRECT":          "Прошло проверку",
		"RejectionRelease": "Выпущено с таможни",
		"SORT":             "Сортировка",
		"DISPATCH":         "Отправлено из центра сортировки",
		"ARRIVE":           "Прибыло в пункт выдачи",
		"DELIVERY":         "Ожидает получателя в пункте выдачи",
		"HAND":             "Вручено",
		"RETURN":           "Возврат",
		"TRANSIT":          "В пути",
	}

	var parts []string
	for _, s := range statuses {
		if t, ok := translations[s]; ok {
			parts = append(parts, t)
		} else {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, ", ")
}

// ─── CDEK Real Client ───────────────────────────────────────────────────

// cdekTrackResponse matches the JSON from cdek.ru/api-site/track/info/
type cdekTrackResponse struct {
	Error  bool   `json:"error"`
	Msg    string `json:"msg"`
	Status []struct {
		Code     string `json:"code"`
		Name     string `json:"name"`
		Date     string `json:"date"` // "27.12.2025"
		CityName string `json:"city_name"`
	} `json:"status"`
	CityFrom string `json:"city_from"`
	CityTo   string `json:"city_to"`
}

// CDEKTracker implements the Tracker interface for CDEK public tracking API.
type CDEKTracker struct {
	client  *http.Client
	baseURL string
}

// NewCDEKTracker creates a new CDEKTracker.
func NewCDEKTracker() *CDEKTracker {
	return &CDEKTracker{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		baseURL: "https://www.cdek.ru/api-site/track/info/",
	}
}

func (c *CDEKTracker) Provider() string { return "CDEK" }

func (c *CDEKTracker) Track(ctx context.Context, trackNumber string) ([]models.TrackingEvent, error) {
	url := fmt.Sprintf("%s?track=%s&locale=ru", c.baseURL, trackNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("cdek: creating request: %w", err)
	}
	req.Header.Set("Accept", "application/json, text/html, */*")
	req.Header.Set("Referer", "https://www.cdek.ru/ru/tracking/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en-US;q=0.8,en;q=0.7")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cdek: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("cdek: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("cdek: reading body: %w", err)
	}

	var data cdekTrackResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("cdek: parsing json: %w", err)
	}

	if data.Error || len(data.Status) == 0 {
		return nil, nil
	}

	var events []models.TrackingEvent
	for _, s := range data.Status {
		eventTime, _ := time.Parse("02.01.2006", s.Date)

		location := s.CityName
		if location == "" {
			location = data.CityFrom + " → " + data.CityTo
		}

		events = append(events, models.TrackingEvent{
			ID:          uuid.New(),
			StatusCode:  s.Code,
			Description: s.Name,
			Location:    location,
			EventTime:   eventTime,
			Source:      "CDEK",
		})
	}

	return events, nil
}
