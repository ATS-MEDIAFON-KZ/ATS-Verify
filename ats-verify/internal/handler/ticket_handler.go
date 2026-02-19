package handler

import (
	"net/http"
	"strings"

	"github.com/google/uuid"

	"ats-verify/internal/middleware"
	"ats-verify/internal/models"
	"ats-verify/internal/service"
)

// TicketHandler handles support ticket endpoints for the Kanban board.
type TicketHandler struct {
	ticketService *service.TicketService
}

// NewTicketHandler creates a new TicketHandler.
func NewTicketHandler(ticketService *service.TicketService) *TicketHandler {
	return &TicketHandler{ticketService: ticketService}
}

// RegisterRoutes registers ticket routes on the mux.
func (h *TicketHandler) RegisterRoutes(mux *http.ServeMux, authMw func(http.Handler) http.Handler) {
	// ATS Staff creates tickets; Customs + Admin can also view/update.
	createMw := middleware.RequireRole(models.RoleATSStaff, models.RoleAdmin)
	viewMw := middleware.RequireRole(models.RoleATSStaff, models.RoleCustoms, models.RoleAdmin)
	updateMw := middleware.RequireRole(models.RoleCustoms, models.RoleAdmin)

	mux.Handle("POST /api/v1/tickets", authMw(createMw(http.HandlerFunc(h.Create))))
	mux.Handle("GET /api/v1/tickets", authMw(viewMw(http.HandlerFunc(h.List))))
	mux.Handle("GET /api/v1/tickets/{id}", authMw(viewMw(http.HandlerFunc(h.GetByID))))
	mux.Handle("PATCH /api/v1/tickets/{id}/status", authMw(updateMw(http.HandlerFunc(h.UpdateStatus))))
	mux.Handle("PATCH /api/v1/tickets/{id}/comment", authMw(viewMw(http.HandlerFunc(h.UpdateComment))))
	mux.Handle("PATCH /api/v1/tickets/{id}/assign", authMw(updateMw(http.HandlerFunc(h.Assign))))
}

// Create handles POST /api/v1/tickets
func (h *TicketHandler) Create(w http.ResponseWriter, r *http.Request) {
	claims := middleware.GetClaims(r)
	if claims == nil {
		Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var input service.CreateTicketInput
	if err := Decode(r, &input); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	createdBy, err := uuid.Parse(claims.UserID)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid user id in token")
		return
	}

	if err := h.ticketService.Create(r.Context(), input, createdBy); err != nil {
		if strings.Contains(err.Error(), "is required") {
			Error(w, http.StatusBadRequest, err.Error())
			return
		}
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusCreated, map[string]string{"message": "ticket created"})
}

// List handles GET /api/v1/tickets?status=to_do
func (h *TicketHandler) List(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	tickets, err := h.ticketService.ListByStatus(r.Context(), status)
	if err != nil {
		if strings.Contains(err.Error(), "invalid status") {
			Error(w, http.StatusBadRequest, err.Error())
			return
		}
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, tickets)
}

// GetByID handles GET /api/v1/tickets/{id}
func (h *TicketHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	ticket, err := h.ticketService.GetByID(r.Context(), id)
	if err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	if ticket == nil {
		Error(w, http.StatusNotFound, "ticket not found")
		return
	}

	JSON(w, http.StatusOK, ticket)
}

// updateStatusRequest is the payload for status change (Kanban drag-and-drop).
type updateStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus handles PATCH /api/v1/tickets/{id}/status
func (h *TicketHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req updateStatusRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.ticketService.UpdateStatus(r.Context(), id, req.Status); err != nil {
		if strings.Contains(err.Error(), "invalid status") {
			Error(w, http.StatusBadRequest, err.Error())
			return
		}
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "status updated"})
}

// updateCommentRequest is the payload for comment update.
type updateCommentRequest struct {
	Field string `json:"field"` // "support_comment" or "customs_comment"
	Value string `json:"value"`
}

// UpdateComment handles PATCH /api/v1/tickets/{id}/comment
func (h *TicketHandler) UpdateComment(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req updateCommentRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.ticketService.UpdateComment(r.Context(), id, req.Field, req.Value); err != nil {
		if strings.Contains(err.Error(), "invalid field") {
			Error(w, http.StatusBadRequest, err.Error())
			return
		}
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "comment updated"})
}

// assignRequest is the payload for assigning a Customs officer.
type assignRequest struct {
	AssigneeID string `json:"assignee_id"`
}

// Assign handles PATCH /api/v1/tickets/{id}/assign
func (h *TicketHandler) Assign(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid ticket id")
		return
	}

	var req assignRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	assigneeID, err := uuid.Parse(req.AssigneeID)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid assignee_id")
		return
	}

	if err := h.ticketService.Assign(r.Context(), id, assigneeID); err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{"message": "ticket assigned"})
}
