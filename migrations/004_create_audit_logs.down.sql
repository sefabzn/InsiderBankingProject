-- Drop indexes
DROP INDEX IF EXISTS idx_audit_logs_entity_type;
DROP INDEX IF EXISTS idx_audit_logs_entity_id;
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_entity_composite;
DROP INDEX IF EXISTS idx_audit_logs_details_gin;

-- Drop audit_logs table
DROP TABLE IF EXISTS audit_logs;
