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

// Create inserts a new support ticket and returns its ID.
func (r *TicketRepository) Create(ctx context.Context, t *models.SupportTicket) (uuid.UUID, error) {
	newID := uuid.New()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO support_tickets
		 (id, iin, full_name, support_ticket_id, application_number, document_number,
		  rejection_reason, attachments, support_comment, customs_comment,
		  status, priority, linked_ticket_id, created_by, assigned_to, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15, NOW(), NOW())`,
		newID, t.IIN, t.FullName, t.SupportTicketID, t.ApplicationNumber,
		t.DocumentNumber, t.RejectionReason, t.Attachments,
		t.SupportComment, t.CustomsComment, t.Status, t.Priority, t.LinkedTicketID,
		t.CreatedBy, t.AssignedTo,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("creating ticket: %w", err)
	}
	return newID, nil
}

// GetByID retrieves a ticket by its UUID.
func (r *TicketRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.SupportTicket, error) {
	var t models.SupportTicket
	err := r.db.QueryRowContext(ctx,
		`SELECT t.id, t.iin, t.full_name, t.support_ticket_id, t.application_number, t.document_number,
		        t.rejection_reason, t.attachments, t.support_comment, t.customs_comment,
		        t.status, t.priority, t.linked_ticket_id, t.created_by, t.assigned_to, t.created_at, t.updated_at,
                r.risk_level, r.comment as risk_comment
		 FROM support_tickets t 
         LEFT JOIN iin_bin_risks r ON t.iin = r.iin_bin 
         WHERE t.id = $1`, id,
	).Scan(
		&t.ID, &t.IIN, &t.FullName, &t.SupportTicketID, &t.ApplicationNumber,
		&t.DocumentNumber, &t.RejectionReason, &t.Attachments,
		&t.SupportComment, &t.CustomsComment, &t.Status, &t.Priority, &t.LinkedTicketID,
		&t.CreatedBy, &t.AssignedTo, &t.CreatedAt, &t.UpdatedAt,
		&t.RiskLevel, &t.RiskComment,
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
	query := `SELECT t.id, t.iin, t.full_name, t.support_ticket_id, t.application_number, t.document_number,
	                  t.rejection_reason, t.attachments, t.support_comment, t.customs_comment,
	                  t.status, t.priority, t.linked_ticket_id, t.created_by, t.assigned_to, t.created_at, t.updated_at,
                      r.risk_level, r.comment as risk_comment
	           FROM support_tickets t
               LEFT JOIN iin_bin_risks r ON t.iin = r.iin_bin`
	args := []interface{}{}

	if status != "" {
		query += " WHERE t.status = $1"
		args = append(args, status)
	}
	query += " ORDER BY CASE t.priority WHEN 'high' THEN 0 WHEN 'medium' THEN 1 WHEN 'low' THEN 2 END, t.created_at DESC"

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
			&t.DocumentNumber, &t.RejectionReason, &t.Attachments,
			&t.SupportComment, &t.CustomsComment, &t.Status, &t.Priority, &t.LinkedTicketID,
			&t.CreatedBy, &t.AssignedTo, &t.CreatedAt, &t.UpdatedAt,
			&t.RiskLevel, &t.RiskComment,
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

// AddAttachments appends new file paths to the attachments array.
func (r *TicketRepository) AddAttachments(ctx context.Context, id uuid.UUID, newPaths []string) error {
	// We use array_cat to combine the old array with the new one, or set it if null
	query := `
		UPDATE support_tickets 
		SET attachments = COALESCE(attachments, '{}'::text[]) || $1, 
		    updated_at = NOW() 
		WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, pq.Array(newPaths), id)
	if err != nil {
		return fmt.Errorf("adding attachments: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("ticket not found")
	}
	return nil
}
