-- Feature 2: Load Risk CSV
CREATE TABLE IF NOT EXISTS risk_raw_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    report_date TIMESTAMP WITH TIME ZONE,
    application_id VARCHAR(100),
    iin_bin VARCHAR(20) NOT NULL,
    document VARCHAR(100),
    user_name VARCHAR(255),
    organization VARCHAR(255),
    status VARCHAR(100),
    reject VARCHAR(100),
    reason TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_risk_raw_data_document ON risk_raw_data(document);
CREATE INDEX IF NOT EXISTS idx_risk_raw_data_iin_bin ON risk_raw_data(iin_bin);

-- Feature 4: Manage Risks
-- Drop old table if needed, or create the new one:
-- (We rename risk_profiles to iin_bin_risks and use RiskLevel ENUM)
DROP TABLE IF EXISTS risk_profiles CASCADE;

CREATE TABLE IF NOT EXISTS iin_bin_risks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    iin_bin VARCHAR(20) UNIQUE NOT NULL,
    risk_level risk_level DEFAULT 'green' NOT NULL,
    flagged_by UUID REFERENCES users(id),
    comment TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_iin_bin_risks_iin_bin ON iin_bin_risks(iin_bin);
