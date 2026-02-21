// User role types mirroring the Go backend
export type UserRole = 'admin' | 'paid_user' | 'ats_staff' | 'customs_staff' | 'marketplace_staff';
export type RiskLevel = 'green' | 'yellow' | 'red';

export interface User {
    id: string;
    username: string;
    role: UserRole;
    marketplace_prefix?: string;
    token: string;
}

export interface Parcel {
    id: string;
    track_number: string;
    marketplace: string;
    country: string;
    brand: string;
    product_name: string;
    snt: string;
    is_used: boolean;
    upload_date: string;
    uploaded_by: string;
    created_at: string;
    updated_at: string;
}

export interface RiskProfile {
    id: string;
    iin_bin: string;
    risk_level: 'red' | 'yellow' | 'green';
    comment: string;
    created_by: string;
    created_at: string;
    updated_at: string;
}

export interface TrackingEvent {
    id: string;
    parcel_id: string;
    status_code: string;
    description: string;
    location: string;
    event_time: string;
    source: string;
}

export interface UploadResult {
    total_processed: number;
    inserted: number;
    updated: number;
    errors: string[];
}

export interface IMEIResult {
    imei: string;
    found_in_pdf: boolean;
}

export interface IMEIReport {
    imei1_found: number;
    imei1_total: number;
    imei2_found: number;
    imei2_total: number;
    imei3_found: number;
    imei3_total: number;
    imei4_found: number;
    imei4_total: number;
    results: IMEIResult[];
}

// Support Tickets (Kanban Board)
export type TicketStatus = 'to_do' | 'in_progress' | 'completed';
export type TicketPriority = 'low' | 'medium' | 'high';

export interface SupportTicket {
    id: string;
    iin: string;
    full_name: string;
    support_ticket_id: string;
    application_number: string;
    document_number: string;
    rejection_reason: string;
    attachments: string[];
    support_comment: string;
    customs_comment: string;
    status: TicketStatus;
    priority: TicketPriority;
    linked_ticket_id?: string;
    created_by: string;
    assigned_to?: string;
    created_at: string;
    updated_at: string;
    risk_level?: RiskLevel;
    risk_comment?: string;
}
