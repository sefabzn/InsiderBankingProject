-- Drop events table
DROP INDEX IF EXISTS idx_events_version;
DROP INDEX IF EXISTS idx_events_created_at;
DROP INDEX IF EXISTS idx_events_type;
DROP INDEX IF EXISTS idx_events_aggregate;
DROP TABLE IF EXISTS events;
