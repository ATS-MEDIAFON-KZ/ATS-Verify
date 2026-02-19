package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// -------------------------------------------------------
// Enums
// -------------------------------------------------------

// UserRole defines the type for user roles.
type UserRole string

const (
	RoleAdmin       UserRole = "admin"
	RolePaidUser    UserRole = "paid_user"
	RoleATSStaff    UserRole = "ats_staff"
	RoleCustoms     UserRole = "customs_staff"
	RoleMarketplace UserRole = "marketplace_staff"
)

// RiskLevel defines the type for IIN/BIN risk assessment.
type RiskLevel string

const (
	RiskGreen  RiskLevel = "green"
	RiskYellow RiskLevel = "yellow"
	RiskRed    RiskLevel = "red"
)

// TicketStatus defines the Kanban column for a support ticket.
type TicketStatus string

const (
	TicketStatusToDo       TicketStatus = "to_do"
	TicketStatusInProgress TicketStatus = "in_progress"
	TicketStatusCompleted  TicketStatus = "completed"
)

// TicketPriority defines urgency level for a support ticket.
type TicketPriority string

const (
	PriorityLow    TicketPriority = "low"
	PriorityMedium TicketPriority = "medium"
	PriorityHigh   TicketPriority = "high"
)

// -------------------------------------------------------
// Domain Models
// -------------------------------------------------------

// User represents a system user.
type User struct {
	ID                uuid.UUID `json:"id" db:"id"`
	Username          string    `json:"username" db:"username"`
	PasswordHash      string    `json:"-" db:"password_hash"`
	Role              UserRole  `json:"role" db:"role"`
	MarketplacePrefix *string   `json:"marketplace_prefix,omitempty" db:"marketplace_prefix"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// Parcel represents a tracked parcel in the system.
type Parcel struct {
	ID          uuid.UUID `json:"id" db:"id"`
	TrackNumber string    `json:"track_number" db:"track_number"`
	Marketplace string    `json:"marketplace" db:"marketplace"`
	Country     string    `json:"country,omitempty" db:"country"`
	Brand       string    `json:"brand,omitempty" db:"brand"`
	ProductName string    `json:"product_name,omitempty" db:"product_name"`
	SNT         string    `json:"snt,omitempty" db:"snt"`
	IsUsed      bool      `json:"is_used" db:"is_used"`
	UploadDate  time.Time `json:"upload_date" db:"upload_date"`
	UploadedBy  uuid.UUID `json:"uploaded_by" db:"uploaded_by"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// RiskProfile represents a risk assessment for an IIN/BIN.
type RiskProfile struct {
	ID        uuid.UUID `json:"id" db:"id"`
	IINBIN    string    `json:"iin_bin" db:"iin_bin"`
	RiskLevel RiskLevel `json:"risk_level" db:"risk_level"`
	FlaggedBy uuid.UUID `json:"flagged_by" db:"flagged_by"`
	Reason    string    `json:"reason,omitempty" db:"reason"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TrackingEvent represents a single tracking event from Kazpost/CDEK.
type TrackingEvent struct {
	ID          uuid.UUID `json:"id" db:"id"`
	ParcelID    uuid.UUID `json:"parcel_id" db:"parcel_id"`
	StatusCode  string    `json:"status_code" db:"status_code"`
	Description string    `json:"description" db:"description"`
	Location    string    `json:"location" db:"location"`
	EventTime   time.Time `json:"event_time" db:"event_time"`
	Source      string    `json:"source" db:"source"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// AnalysisReport stores results of IMEI verification or risk analysis.
type AnalysisReport struct {
	ID            uuid.UUID              `json:"id" db:"id"`
	UserID        uuid.UUID              `json:"user_id" db:"user_id"`
	ReportType    string                 `json:"report_type" db:"report_type"`
	InputFileName string                 `json:"input_file_name" db:"input_file_name"`
	ResultSummary map[string]interface{} `json:"result_summary" db:"result_summary"`
	RawDataURL    string                 `json:"raw_data_url,omitempty" db:"raw_data_url"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

// SupportTicket represents a Kanban board ticket for the ATS → Customs workflow.
// ATS Staff creates tickets for rejected applications; Customs moves/resolves them.
type SupportTicket struct {
	ID                uuid.UUID      `json:"id" db:"id"`
	IIN               string         `json:"iin" db:"iin"`
	FullName          string         `json:"full_name" db:"full_name"`
	SupportTicketID   string         `json:"support_ticket_id" db:"support_ticket_id"`
	ApplicationNumber string         `json:"application_number" db:"application_number"`
	DocumentNumber    string         `json:"document_number" db:"document_number"`
	RejectionReason   string         `json:"rejection_reason" db:"rejection_reason"`
	Attachments       pq.StringArray `json:"attachments" db:"attachments"`
	SupportComment    string         `json:"support_comment" db:"support_comment"`
	CustomsComment    string         `json:"customs_comment" db:"customs_comment"`
	Status            TicketStatus   `json:"status" db:"status"`
	Priority          TicketPriority `json:"priority" db:"priority"`
	CreatedBy         uuid.UUID      `json:"created_by" db:"created_by"`
	AssignedTo        *uuid.UUID     `json:"assigned_to,omitempty" db:"assigned_to"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at" db:"updated_at"`
}

// MarketplacePrefixMap maps user role suffixes to marketplace names.
var MarketplacePrefixMap = map[string]string{
	"wb":    "Wildberries",
	"ozon":  "Ozon",
	"kaspi": "Kaspi",
	"ali":   "AliExpress",
	"temu":  "Temu",
}

// -------------------------------------------------------
// IMEI Verification Report (output format)
// -------------------------------------------------------

// IMEIMatchResult represents the verification result for a single IMEI value.
type IMEIMatchResult struct {
	CSVLine     int    `json:"csv_line"`               // 1-based row number in the source CSV
	Column      string `json:"column"`                 // Column name, e.g. "Imei1", "Imei2"
	IMEI14      string `json:"imei_14"`                // 14-digit IMEI from CSV (without Luhn check digit)
	MatchedIMEI string `json:"matched_imei,omitempty"` // 15-digit sequence found in PDF (if matched)
	Found       bool   `json:"found"`                  // Whether the 14-digit prefix was found inside PDF
}

// IMEIColumnStats holds per-column statistics (e.g. stats for "Imei1", "Imei2", etc.).
type IMEIColumnStats struct {
	Column  string `json:"column"`
	Total   int    `json:"total"`
	Found   int    `json:"found"`
	Missing int    `json:"missing"`
}

// IMEIVerificationReport is the full output of an IMEI-vs-PDF verification job.
// Designed per GOALS.md spec: top stats → per-column breakdown → line-by-line results.
type IMEIVerificationReport struct {
	// Aggregate totals
	TotalIMEIs   int `json:"total_imeis"`
	TotalFound   int `json:"total_found"`
	TotalMissing int `json:"total_missing"`

	// Per-column breakdown (Imei1, Imei2, Imei3, Imei4)
	ColumnStats []IMEIColumnStats `json:"column_stats"`

	// Line-by-line verification results
	Results []IMEIMatchResult `json:"results"`
}
