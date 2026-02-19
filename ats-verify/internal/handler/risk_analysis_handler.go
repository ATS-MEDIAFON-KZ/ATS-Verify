package handler

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"ats-verify/internal/middleware"
	"ats-verify/internal/models"
	"ats-verify/internal/service"
)

// RiskAnalysisHandler handles the advanced risk analysis CSV upload endpoint.
type RiskAnalysisHandler struct {
	riskAnalysisService *service.RiskAnalysisService
}

// NewRiskAnalysisHandler creates a new RiskAnalysisHandler.
func NewRiskAnalysisHandler(riskAnalysisService *service.RiskAnalysisService) *RiskAnalysisHandler {
	return &RiskAnalysisHandler{riskAnalysisService: riskAnalysisService}
}

// RegisterRoutes registers risk analysis routes.
func (h *RiskAnalysisHandler) RegisterRoutes(mux *http.ServeMux, authMw func(http.Handler) http.Handler) {
	roleMw := middleware.RequireRole(models.RoleAdmin)
	mux.Handle("POST /api/v1/risks/analyze", authMw(roleMw(http.HandlerFunc(h.AnalyzeCSV))))
}

// AnalyzeCSV handles POST /api/v1/risks/analyze (multipart: csv_file)
// Parses a CSV with application data and runs risk detection algorithms.
func (h *RiskAnalysisHandler) AnalyzeCSV(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		Error(w, http.StatusBadRequest, "failed to parse form: "+err.Error())
		return
	}

	csvFile, _, err := r.FormFile("csv_file")
	if err != nil {
		Error(w, http.StatusBadRequest, "csv_file is required")
		return
	}
	defer csvFile.Close()

	flaggedBy, err := uuid.Parse(claims.UserID)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid user id in token")
		return
	}

	result, err := h.riskAnalysisService.AnalyzeCSV(r.Context(), csvFile, flaggedBy)
	if err != nil {
		if strings.Contains(err.Error(), "missing required column") || strings.Contains(err.Error(), "no valid data") {
			Error(w, http.StatusBadRequest, err.Error())
			return
		}
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, result)
}
