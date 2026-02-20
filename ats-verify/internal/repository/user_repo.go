package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"

	"ats-verify/internal/models"
)

// UserRepository handles user database operations.
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create inserts a new user into the database.
func (r *UserRepository) Create(ctx context.Context, u *models.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash, role, marketplace_prefix, is_approved, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())`,
		uuid.New(), u.Username, u.PasswordHash, u.Role, u.MarketplacePrefix, u.IsApproved,
	)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}
	return nil
}

// GetByUsername retrieves a user by username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var u models.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, marketplace_prefix, is_approved, created_at, updated_at
		 FROM users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MarketplacePrefix, &u.IsApproved, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying user by username: %w", err)
	}
	return &u, nil
}

// GetByID retrieves a user by ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var u models.User
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash, role, marketplace_prefix, is_approved, created_at, updated_at
		 FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.MarketplacePrefix, &u.IsApproved, &u.CreatedAt, &u.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying user by id: %w", err)
	}
	return &u, nil
}

// ApproveUser sets is_approved = true for the given user ID.
func (r *UserRepository) ApproveUser(ctx context.Context, id uuid.UUID) error {
	res, err := r.db.ExecContext(ctx, `UPDATE users SET is_approved = true, updated_at = NOW() WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("approving user %s: %w", id, err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("user %s not found", id)
	}
	return nil
}
