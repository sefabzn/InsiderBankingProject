-- Create events table for event sourcing
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type VARCHAR(50) NOT NULL, -- 'user', 'balance', 'transaction'
    aggregate_id UUID NOT NULL,
    event_type VARCHAR(100) NOT NULL, -- 'UserRegistered', 'AmountCredited', etc.
    event_data JSONB NOT NULL, -- Event payload
    event_metadata JSONB, -- Optional metadata (correlation_id, etc.)
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER NOT NULL DEFAULT 1 -- For optimistic concurrency
);

-- Index for efficient querying by aggregate
CREATE INDEX idx_events_aggregate ON events(aggregate_type, aggregate_id);

-- Index for event type queries
CREATE INDEX idx_events_type ON events(event_type);

-- Index for time-based queries
CREATE INDEX idx_events_created_at ON events(created_at);

-- Index for version ordering within aggregate
CREATE INDEX idx_events_version ON events(aggregate_type, aggregate_id, version);
