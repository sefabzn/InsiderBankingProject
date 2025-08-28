-- Create audit_logs table
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(100) NOT NULL,
    details JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add constraints for valid entity types
ALTER TABLE audit_logs ADD CONSTRAINT chk_audit_logs_entity_type 
    CHECK (entity_type IN ('user', 'transaction', 'balance'));

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_type ON audit_logs(entity_type);
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_id ON audit_logs(entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_action ON audit_logs(action);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_audit_logs_entity_composite ON audit_logs(entity_type, entity_id, created_at DESC);

-- GIN index for JSONB details column for efficient querying
CREATE INDEX IF NOT EXISTS idx_audit_logs_details_gin ON audit_logs USING gin(details);
