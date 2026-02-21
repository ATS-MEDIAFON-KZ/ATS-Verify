package service

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

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
	LinkedTicketID    *string  `json:"linked_ticket_id,omitempty"`
}

// Create validates and creates a new support ticket, returning its ID.
func (s *TicketService) Create(ctx context.Context, input CreateTicketInput, createdBy uuid.UUID) (uuid.UUID, error) {
	// Validate required fields.
	if input.IIN == "" {
		return uuid.Nil, fmt.Errorf("iin is required")
	}
	if input.FullName == "" {
		return uuid.Nil, fmt.Errorf("full_name is required")
	}
	if input.SupportTicketID == "" {
		return uuid.Nil, fmt.Errorf("support_ticket_id is required")
	}
	if input.ApplicationNumber == "" {
		return uuid.Nil, fmt.Errorf("application_number is required")
	}
	if input.DocumentNumber == "" {
		return uuid.Nil, fmt.Errorf("document_number is required")
	}
	if input.RejectionReason == "" {
		return uuid.Nil, fmt.Errorf("rejection_reason is required")
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

// AddAttachments handles saving files to disk and updating the ticket's attachments list.
func (s *TicketService) AddAttachments(ctx context.Context, id uuid.UUID, files []*multipart.FileHeader) error {
	uploadDir := filepath.Join("uploads", "tickets", id.String())
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return fmt.Errorf("failed to create upload directory: %w", err)
	}

	var paths []string
	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			return fmt.Errorf("failed to open uploaded file: %w", err)
		}
		defer file.Close()

		filename := filepath.Base(fileHeader.Filename)
		destPath := filepath.Join(uploadDir, filename)

		dest, err := os.Create(destPath)
		if err != nil {
			return fmt.Errorf("failed to create destination file: %w", err)
		}
		defer dest.Close()

		if _, err := io.Copy(dest, file); err != nil {
			return fmt.Errorf("failed to copy file contents: %w", err)
		}

		// Save the relative URL path to serve statically
		relativePath := fmt.Sprintf("/api/v1/attachments/tickets/%s/%s", id.String(), filename)
		paths = append(paths, relativePath)
	}

	return s.ticketRepo.AddAttachments(ctx, id, paths)
}
