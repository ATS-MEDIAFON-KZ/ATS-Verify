package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"ats-verify/internal/middleware"
	"ats-verify/internal/models"
	"ats-verify/internal/repository"
)

// AuthService handles authentication logic.
type AuthService struct {
	userRepo  *repository.UserRepository
	jwtSecret string
	jwtExpiry time.Duration
}

// NewAuthService creates a new AuthService.
func NewAuthService(userRepo *repository.UserRepository, jwtSecret string, jwtExpiry time.Duration) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		jwtSecret: jwtSecret,
		jwtExpiry: jwtExpiry,
	}
}

// LoginResponse is returned on successful login.
type LoginResponse struct {
	Token string      `json:"token"`
	User  models.User `json:"user"`
}

// Login validates credentials and returns a JWT token.
func (s *AuthService) Login(ctx context.Context, username, password string) (*LoginResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("querying user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.IsApproved {
		return nil, fmt.Errorf("user account is pending admin approval")
	}

	// Generate JWT
	prefix := ""
	if user.MarketplacePrefix != nil {
		prefix = *user.MarketplacePrefix
	}

	claims := &middleware.Claims{
		UserID:            user.ID.String(),
		Username:          user.Username,
		Role:              user.Role,
		MarketplacePrefix: prefix,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.jwtExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("signing token: %w", err)
	}

	return &LoginResponse{
		Token: tokenStr,
		User:  *user,
	}, nil
}

// HashPassword creates a bcrypt hash from a plain-text password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(hash), nil
}

// Register creates a new user. Auto-approves "@ats-mediafon.kz" and sets them to RoleATSStaff.
func (s *AuthService) Register(ctx context.Context, username, password string) (*models.User, error) {
	// Check if user already exists
	existing, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("checking existing user: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("username already taken")
	}

	hash, err := HashPassword(password)
	if err != nil {
		return nil, err
	}

	// Determine defaults
	isApproved := false
	role := models.RoleMarketplace // Default role assuming mostly marketplaces or customs register this way, but typically we want Admin to assign correct role or use simple default
	var prefix *string

	emailLower := strings.ToLower(username)
	if emailLower == "q.aldaniyazov@ats-mediafon.kz" || emailLower == "y.sedochenko@ats-mediafon.kz" {
		isApproved = true
		role = models.RoleAdmin
	} else if strings.HasSuffix(emailLower, "@ats-mediafon.kz") {
		isApproved = true
		role = models.RoleATSStaff
	}

	if !isApproved {
		role = models.RoleMarketplace
	}

	user := &models.User{
		Username:          username,
		PasswordHash:      hash,
		Role:              role,
		IsApproved:        isApproved,
		MarketplacePrefix: prefix,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	// Fetch to return full model with ID
	created, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("retrieving created user: %w", err)
	}

	return created, nil
}

// ApproveUser marks a user as approved.
func (s *AuthService) ApproveUser(ctx context.Context, userID uuid.UUID) error {
	return s.userRepo.ApproveUser(ctx, userID)
}
