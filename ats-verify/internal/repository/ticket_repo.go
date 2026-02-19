package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"ats-verify/internal/models"
)

// TicketRepository handles support ticket database operations.
type TicketRepository struct {
	db *sql.DB
}

// NewTicketRepository creates a new TicketRepository.
func NewTicketRepository(db *sql.DB) *TicketRepository {
	return &TicketRepository{db: db}
}

// Create inserts a new support ticket.
func (r *TicketRepository) Create(ctx context.Context, t *models.SupportTicket) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO support_tickets
		 (id, iin, full_name, support_ticket_id, application_number, document_number,
		  rejection_reason, attachments, support_comment, customs_comment,
		  status, priority, created_by, assigned_to, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14, NOW(), NOW())`,
		uuid.New(), t.IIN, t.FullName, t.SupportTicketID, t.ApplicationNumber,
		t.DocumentNumber, t.RejectionReason, pq.Array(t.Attachments),
		t.SupportComment, t.CustomsComment, t.Status, t.Priority,
		t.CreatedBy, t.AssignedTo,
	)
	if err != nil {
		return fmt.Errorf("creating ticket: %w", err)
	}
	return nil
}

// GetByID retrieves a ticket by its UUID.
func (r *TicketRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SupportTicket, error) {
	var t models.SupportTicket
	err := r.db.QueryRowContext(ctx,
		`SELECT id, iin, full_name, support_ticket_id, application_number, document_number,
		        rejection_reason, attachments, support_comment, customs_comment,
		        status, priority, created_by, assigned_to, created_at, updated_at
		 FROM support_tickets WHERE id = $1`, id,
	).Scan(
		&t.ID, &t.IIN, &t.FullName, &t.SupportTicketID, &t.ApplicationNumber,
		&t.DocumentNumber, &t.RejectionReason, pq.Array(&t.Attachments),
		&t.SupportComment, &t.CustomsComment, &t.Status, &t.Priority,
		&t.CreatedBy, &t.AssignedTo, &t.CreatedAt, &t.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying ticket by id: %w", err)
	}
	return &t, nil
}

// ListByStatus returns tickets filtered by optional status, sorted for Kanban view.
func (r *TicketRepository) ListByStatus(ctx context.Context, status string) ([]models.SupportTicket, error) {
	query := `SELECT id, iin, full_name, support_ticket_id, application_number, document_number,
	                  rejection_reason, attachments, support_comment, customs_comment,
	                  status, priority, created_by, assigned_to, created_at, updated_at
	           FROM support_tickets`
	args := []interface{}{}

	if status != "" {
		query += " WHERE status = $1"
		args = append(args, status)
	}
	query += " ORDER BY CASE priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 WHEN 'low' THEN 2 END, created_at DESC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listing tickets: %w", err)
	}
	defer rows.Close()

	var tickets []models.SupportTicket
	for rows.Next() {
		var t models.SupportTicket
		if err := rows.Scan(
			&t.ID, &t.IIN, &t.FullName, &t.SupportTicketID, &t.ApplicationNumber,
			&t.DocumentNumber, &t.RejectionReason, pq.Array(&t.Attachments),
			&t.SupportComment, &t.CustomsComment, &t.Status, &t.Priority,
			&t.CreatedBy, &t.AssignedTo, &t.CreatedAt, &t.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scanning ticket row: %w", err)
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// UpdateStatus changes the Kanban column for a ticket (drag-and-drop).
func (r *TicketRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.TicketStatus) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE support_tickets SET status = $1, updated_at = NOW() WHERE id = $2",
		status, id,
	)
	if err != nil {
		return fmt.Errorf("updating ticket status: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("ticket not found")
	}
	return nil
}

// UpdateComment updates either support_comment or customs_comment.
func (r *TicketRepository) UpdateComment(ctx context.Context, id uuid.UUID, field, value string) error {
	// Whitelist allowed fields to prevent SQL injection.
	var col string
	switch field {
	case "support_comment":
		col = "support_comment"
	case "customs_comment":
		col = "customs_comment"
	default:
		return fmt.Errorf("invalid comment field: %s", field)
	}

	query := fmt.Sprintf("UPDATE support_tickets SET %s = $1, updated_at = NOW() WHERE id = $2", col)
	result, err := r.db.ExecContext(ctx, query, value, id)
	if err != nil {
		return fmt.Errorf("updating ticket comment: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("ticket not found")
	}
	return nil
}

// AssignTo assigns a Customs officer to a ticket.
func (r *TicketRepository) AssignTo(ctx context.Context, id, assigneeID uuid.UUID) error {
	result, err := r.db.ExecContext(ctx,
		"UPDATE support_tickets SET assigned_to = $1, updated_at = NOW() WHERE id = $2",
		assigneeID, id,
	)
	if err != nil {
		return fmt.Errorf("assigning ticket: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("ticket not found")
	}
	return nil
}
