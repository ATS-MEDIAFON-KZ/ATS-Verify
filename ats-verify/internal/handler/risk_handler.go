package handler

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"ats-verify/internal/middleware"
	"ats-verify/internal/models"
	"ats-verify/internal/service"
)

// RiskHandler handles risk profile endpoints.
type RiskHandler struct {
	riskService *service.RiskService
}

// NewRiskHandler creates a new RiskHandler.
func NewRiskHandler(riskService *service.RiskService) *RiskHandler {
	return &RiskHandler{riskService: riskService}
}

// RegisterRoutes registers risk routes.
func (h *RiskHandler) RegisterRoutes(mux *http.ServeMux, authMw func(http.Handler) http.Handler) {
	roleMw := middleware.RequireRole(models.RoleAdmin, models.RoleCustoms)

	mux.Handle("GET /api/v1/risks", authMw(roleMw(http.HandlerFunc(h.List))))
	mux.Handle("POST /api/v1/risks", authMw(roleMw(http.HandlerFunc(h.CreateOrUpdate))))
	mux.Handle("DELETE /api/v1/risks/{id}", authMw(roleMw(http.HandlerFunc(h.Delete))))
}

// List handles GET /api/v1/risks
func (h *RiskHandler) List(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.riskService.List(r.Context())
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	if profiles == nil {
		profiles = []models.IINBINRisk{}
	}

	JSON(w, http.StatusOK, map[string]interface{}{
		"profiles": profiles,
		"total":    len(profiles),
	})
}

// createRiskRequest is the payload for creating/updating a risk profile.
type createRiskRequest struct {
	IINBIN    string `json:"iin_bin"`
	RiskLevel string `json:"risk_level"`
	Comment   string `json:"comment"`
}

// CreateOrUpdate handles POST /api/v1/risks
func (h *RiskHandler) CreateOrUpdate(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req createRiskRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.IINBIN == "" || req.RiskLevel == "" {
		Error(w, http.StatusBadRequest, "iin_bin and risk_level are required")
		return
	}

	if strings.TrimSpace(req.Comment) == "" {
		Error(w, http.StatusBadRequest, "comment is strictly required")
		return
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		Error(w, http.StatusInternalServerError, "invalid user ID")
		return
	}

	if err := h.riskService.CreateOrUpdate(r.Context(), req.IINBIN, models.RiskLevel(req.RiskLevel), req.Comment, userID); err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "risk profile saved"})
}

// Delete handles DELETE /api/v1/risks/{id}
func (h *RiskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	if err := h.riskService.Delete(r.Context(), id); err != nil {
		Error(w, http.StatusNotFound, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "risk profile deleted"})
}
