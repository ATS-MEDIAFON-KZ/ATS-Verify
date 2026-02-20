package handler

import (
	"net/http"

	"github.com/google/uuid"

	"ats-verify/internal/middleware"
	"ats-verify/internal/models"
	"ats-verify/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService *service.AuthService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

// RegisterRoutes registers auth routes on the mux.
func (h *AuthHandler) RegisterRoutes(mux *http.ServeMux, authMw func(http.Handler) http.Handler) {
	mux.HandleFunc("POST /api/v1/auth/login", h.Login)
	mux.HandleFunc("POST /api/v1/auth/register", h.Register)
	mux.Handle("POST /api/admin/users/{id}/approve", authMw(
		middleware.RequireRole(models.RoleAdmin)(http.HandlerFunc(h.ApproveUser)),
	))
}

// loginRequest is the expected login payload.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Login handles POST /api/v1/auth/login
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		Error(w, http.StatusBadRequest, "username and password are required")
		return
	}

	resp, err := h.authService.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	JSON(w, http.StatusOK, resp)
}

// registerRequest is the expected registration payload.
type registerRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Register handles POST /api/v1/auth/register
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := Decode(r, &req); err != nil {
		Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Username == "" || req.Password == "" {
		Error(w, http.StatusBadRequest, "username and password are required")
		return
	}

	user, err := h.authService.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		Error(w, http.StatusBadRequest, err.Error())
		return
	}

	// Based on requirement: non-approved users need manual approval
	message := "Registration successful. Pending admin approval."
	if user.IsApproved {
		message = "Registration successful. Account approved."
	}

	JSON(w, http.StatusCreated, map[string]interface{}{
		"message": message,
		"user":    user,
	})
}

// ApproveUser handles POST /api/admin/users/{id}/approve
func (h *AuthHandler) ApproveUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	if err := h.authService.ApproveUser(r.Context(), userID); err != nil {
		Error(w, http.StatusInternalServerError, err.Error())
		return
	}

	JSON(w, http.StatusOK, map[string]string{
		"message": "User approved successfully",
	})
}
