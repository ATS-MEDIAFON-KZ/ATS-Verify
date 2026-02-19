package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"ats-verify/internal/models"
	"ats-verify/internal/repository"
)

// TicketService handles support ticket business logic.
type TicketService struct {
	ticketRepo *repository.TicketRepository
}

// NewTicketService creates a new TicketService.
func NewTicketService(ticketRepo *repository.TicketRepository) *TicketService {
	return &TicketService{ticketRepo: ticketRepo}
}

// CreateTicketInput is the validated input for creating a ticket.
type CreateTicketInput struct {
	IIN               string   `json:"iin"`
	FullName          string   `json:"full_name"`
	SupportTicketID   string   `json:"support_ticket_id"`
	ApplicationNumber string   `json:"application_number"`
	DocumentNumber    string   `json:"document_number"`
	RejectionReason   string   `json:"rejection_reason"`
	Attachments       []string `json:"attachments"`
	SupportComment    string   `json:"support_comment"`
	Priority          string   `json:"priority"`
}

// Create validates and creates a new support ticket.
func (s *TicketService) Create(ctx context.Context, input CreateTicketInput, createdBy uuid.UUID) error {
	// Validate required fields.
	if input.IIN == "" {
		return fmt.Errorf("iin is required")
	}
	if input.FullName == "" {
		return fmt.Errorf("full_name is required")
	}
	if input.SupportTicketID == "" {
		return fmt.Errorf("support_ticket_id is required")
	}
	if input.ApplicationNumber == "" {
		return fmt.Errorf("application_number is required")
	}
	if input.DocumentNumber == "" {
		return fmt.Errorf("document_number is required")
	}
	if input.RejectionReason == "" {
		return fmt.Errorf("rejection_reason is required")
	}

	priority := models.PriorityMedium
	switch models.TicketPriority(input.Priority) {
	case models.PriorityLow, models.PriorityMedium, models.PriorityHigh:
		priority = models.TicketPriority(input.Priority)
	}

	ticket := &models.SupportTicket{
		IIN:               input.IIN,
		FullName:          input.FullName,
		SupportTicketID:   input.SupportTicketID,
		ApplicationNumber: input.ApplicationNumber,
		DocumentNumber:    input.DocumentNumber,
		RejectionReason:   input.RejectionReason,
		Attachments:       input.Attachments,
		SupportComment:    input.SupportComment,
		Status:            models.TicketStatusToDo,
		Priority:          priority,
		CreatedBy:         createdBy,
	}

	return s.ticketRepo.Create(ctx, ticket)
}

// GetByID retrieves a single ticket.
func (s *TicketService) GetByID(ctx context.Context, id uuid.UUID) (*models.SupportTicket, error) {
	return s.ticketRepo.GetByID(ctx, id)
}

// ListByStatus lists tickets, optionally filtered by Kanban column.
func (s *TicketService) ListByStatus(ctx context.Context, status string) ([]models.SupportTicket, error) {
	// Validate status if provided.
	if status != "" {
		switch models.TicketStatus(status) {
		case models.TicketStatusToDo, models.TicketStatusInProgress, models.TicketStatusCompleted:
		default:
			return nil, fmt.Errorf("invalid status: %s (allowed: to_do, in_progress, completed)", status)
		}
	}
	return s.ticketRepo.ListByStatus(ctx, status)
}

// UpdateStatus changes the Kanban column (drag-and-drop action by Customs).
func (s *TicketService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	ts := models.TicketStatus(status)
	switch ts {
	case models.TicketStatusToDo, models.TicketStatusInProgress, models.TicketStatusCompleted:
	default:
		return fmt.Errorf("invalid status: %s", status)
	}
	return s.ticketRepo.UpdateStatus(ctx, id, ts)
}

// UpdateComment updates a comment field on a ticket.
func (s *TicketService) UpdateComment(ctx context.Context, id uuid.UUID, field, value string) error {
	if field != "support_comment" && field != "customs_comment" {
		return fmt.Errorf("invalid field: %s (allowed: support_comment, customs_comment)", field)
	}
	return s.ticketRepo.UpdateComment(ctx, id, field, value)
}

// Assign assigns a Customs officer to a ticket.
func (s *TicketService) Assign(ctx context.Context, id, assigneeID uuid.UUID) error {
	return s.ticketRepo.AssignTo(ctx, id, assigneeID)
}
