-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 1. Enums for Role-Based Access Control and Statuses
CREATE TYPE user_role AS ENUM ('admin', 'paid_user', 'ats_staff', 'customs_staff', 'marketplace_staff');
CREATE TYPE risk_level AS ENUM ('green', 'yellow', 'red');

-- 2. Users Table
-- Supports all roles. 'marketplace_prefix' is used for 'marketplace_staff' to determine the source (e.g., 'wb').
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role user_role NOT NULL,
    marketplace_prefix VARCHAR(50), -- Nullable, only for marketplace_staff
    is_approved BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 3. Risk Profiles (IIN/BIN)
CREATE TABLE iin_bin_risks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    iin_bin VARCHAR(20) UNIQUE NOT NULL,
    risk_level risk_level DEFAULT 'green',
    flagged_by UUID REFERENCES users(id),
    comment TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Risk Raw Data for ingestion
CREATE TABLE risk_raw_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    report_date TIMESTAMP WITH TIME ZONE NOT NULL,
    application_id VARCHAR(100) NOT NULL,
    iin_bin VARCHAR(20) NOT NULL,
    document VARCHAR(100) NOT NULL,
    user_name VARCHAR(255) NOT NULL,
    organization VARCHAR(255),
    status VARCHAR(50),
    reject VARCHAR(255),
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 4. Parcels Table
-- Core logic: 'track_number' is unique. 'is_used' flag determines if Customs/ATS has processed it.
CREATE TABLE parcels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    track_number VARCHAR(100) UNIQUE NOT NULL,
    marketplace VARCHAR(50) NOT NULL, -- Derived from user prefix or token
    country VARCHAR(50),
    brand VARCHAR(100),
    product_name TEXT,
    snt VARCHAR(100), -- Consignment Note / SNT
    
    -- Status Logic:
    -- FALSE = Not Used (default on insert)
    -- TRUE = Used (marked by Customs/ATS)
    is_used BOOLEAN DEFAULT FALSE,
    
    upload_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(), -- The date from the CSV
    uploaded_by UUID REFERENCES users(id),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for fast lookup by track number
CREATE INDEX idx_parcels_track_number ON parcels(track_number);
CREATE INDEX idx_parcels_is_used ON parcels(is_used);

-- 5. Tracking Events
-- Stores history from Kazpost/CDEK.
CREATE TABLE tracking_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parcel_id UUID REFERENCES parcels(id) ON DELETE CASCADE,
    status_code VARCHAR(50),
    description TEXT,
    location VARCHAR(100),
    event_time TIMESTAMP WITH TIME ZONE,
    source VARCHAR(50), -- 'Kazpost', 'CDEK'
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- 6. Analysis Reports
-- distinct from tracking, this stores the result of "IMEI vs PDF" or "Risk Analysis" jobs.
CREATE TABLE analysis_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    report_type VARCHAR(50), -- 'imei_verification', 'customs_risk'
    input_file_name VARCHAR(255),
    result_summary JSONB, -- Stores JSON summary like {"found": 10, "total": 12}
    raw_data_url TEXT, -- Path to result CSV/File if stored
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================
-- 7. Support Tickets (Kanban Board: ATS → Customs Workflow)
-- ============================================================

CREATE TYPE ticket_status AS ENUM ('to_do', 'in_progress', 'completed');
CREATE TYPE ticket_priority AS ENUM ('low', 'medium', 'high');

-- Replaces legacy Google Sheet for rejected application appeals.
-- ATS Staff creates tickets; Customs officers move them across Kanban columns.
CREATE TABLE support_tickets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Applicant info
    iin VARCHAR(20) NOT NULL,
    full_name VARCHAR(255) NOT NULL,

    -- External reference IDs
    support_ticket_id VARCHAR(100) UNIQUE NOT NULL,  -- ID заявки в СП
    application_number VARCHAR(100) NOT NULL,         -- Номер заявки
    document_number VARCHAR(100) NOT NULL,            -- Номер документа

    -- Content
    rejection_reason TEXT NOT NULL,                    -- Причина отклонения
    attachments TEXT[] DEFAULT '{}',                   -- Подтверждающие документы (URLs)
    support_comment TEXT DEFAULT '',                   -- Комментарий СП (ATS Staff edits)
    customs_comment TEXT DEFAULT '',                   -- Комментарий ГО (Customs edits upon resolution)

    -- Kanban state
    status ticket_status NOT NULL DEFAULT 'to_do',
    priority ticket_priority NOT NULL DEFAULT 'medium',

    -- Ownership
    created_by UUID NOT NULL REFERENCES users(id),    -- ATS Staff who created the ticket
    assigned_to UUID REFERENCES users(id),            -- Customs officer (nullable until assigned)

    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for Kanban board queries
CREATE INDEX idx_support_tickets_status ON support_tickets(status);
CREATE INDEX idx_support_tickets_iin ON support_tickets(iin);
CREATE INDEX idx_support_tickets_assigned_to ON support_tickets(assigned_to);
