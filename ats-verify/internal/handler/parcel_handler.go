package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"ats-verify/internal/middleware"
	"ats-verify/internal/models"
	"ats-verify/internal/service"
)

// ParcelHandler handles parcel CRUD endpoints.
type ParcelHandler struct {
	parcelService *service.ParcelService
}

// NewParcelHandler creates a new ParcelHandler.
func NewParcelHandler(parcelService *service.ParcelService) *ParcelHandler {
	return &ParcelHandler{parcelService: parcelService}
}

// RegisterRoutes registers parcel routes (must be wrapped with auth middleware).
func (h *ParcelHandler) RegisterRoutes(mux *http.ServeMux, authMw func(http.Handler) http.Handler) {
	mux.Handle("GET /api/v1/parcels", authMw(http.HandlerFunc(h.List)))
	mux.Handle("POST /api/v1/parcels/upload", authMw(
		middleware.RequireRole(models.RoleMarketplace, models.RoleAdmin)(http.HandlerFunc(h.Upload)),
	))
	mux.Handle("POST /api/v1/parcels/upload-json", authMw(
		middleware.RequireRole(models.RoleMarketplace, models.RoleAdmin)(http.HandlerFunc(h.UploadJSON)),
	))
	mux.Handle("POST /api/v1/parcels/mark-used", authMw(
		middleware.RequireRole(models.RoleCustoms)(http.HandlerFunc(h.MarkUsed)),
	))
}

// List handles GET /api/v1/parcels?status=&search=&page=&limit=
func (h *ParcelHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))

	filter := service.ListParcelsFilter{
		Status: q.Get("status"),
		Search: q.Get("search"),
		Page:   page,
		Limit:  limit,
	}

	resp, err := h.parcelService.ListParcels(r.Context(), filter)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, resp)
}

// Upload handles POST /api/v1/parcels/upload (multipart/form-data)
func (h *ParcelHandler) Upload(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	// Parse multipart form (max 10MB)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		Error(w, http.StatusBadRequest, "failed to parse form: "+err.Error())
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		Error(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	// Determine marketplace from form value or user role
	overrideMarketplace := strings.TrimSpace(r.FormValue("marketplace"))

	if claims.Role == models.RoleMarketplace {
		if claims.MarketplacePrefix != "" {
			if name, ok := models.MarketplacePrefixMap[claims.MarketplacePrefix]; ok {
				overrideMarketplace = name
			} else {
				overrideMarketplace = claims.MarketplacePrefix
			}
		} else if overrideMarketplace == "" {
			overrideMarketplace = "Unknown Marketplace"
		}
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "invalid user ID in token")
		return
	}

	result, err := h.parcelService.ProcessCSVUpload(r.Context(), file, overrideMarketplace, userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, result)
}

// markUsedRequest is the payload for marking a parcel as used.
type markUsedRequest struct {
	TrackNumber string `json:"track_number"`
}

// MarkUsed handles POST /api/v1/parcels/mark-used
func (h *ParcelHandler) MarkUsed(w http.ResponseWriter, r *http.Request) {
	var req struct {
		TrackNumber string `json:"track_number"`
	}
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	track := strings.TrimSpace(req.TrackNumber)
	if track == "" {
		Error(w, http.StatusBadRequest, "track_number is required")
		return
	}

	// Обновляем статус посылки. Ошибка "not found" пробрасывается из repo
	if err := h.parcelService.MarkParcelUsed(r.Context(), track); err != nil {
		if strings.Contains(err.Error(), "not found") {
			Error(w, http.StatusNotFound, "Трек-номер не найден в базе данных")
			return
		}
		Error(w, http.StatusInternalServerError, "failed to update database")
		return
	}

	JSON(w, http.StatusOK, map[string]string{
		"message":      "parcel marked as used",
		"track_number": track,
	})
}

// UploadJSON handles POST /api/v1/parcels/upload-json (application/json)
func (h *ParcelHandler) UploadJSON(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var reqs []service.JSONUploadRequest
	if err := Decode(r, &reqs); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	overrideMarketplace := ""
	if claims.Role == models.RoleMarketplace {
		if claims.MarketplacePrefix != "" {
			if name, ok := models.MarketplacePrefixMap[claims.MarketplacePrefix]; ok {
				overrideMarketplace = name
			} else {
				overrideMarketplace = claims.MarketplacePrefix
			}
		} else {
			overrideMarketplace = "Unknown Marketplace"
		}
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "invalid user ID in token")
		return
	}

	result, err := h.parcelService.ProcessJSONUpload(r.Context(), reqs, overrideMarketplace, userID)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, result)
}
