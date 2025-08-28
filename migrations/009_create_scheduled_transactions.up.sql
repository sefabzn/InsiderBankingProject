-- Create scheduled_transactions table for recurring and scheduled transactions
CREATE TABLE scheduled_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('credit', 'debit', 'transfer')),

    -- Transaction details
    amount DECIMAL(15,2) NOT NULL CHECK (amount > 0),
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    description TEXT,

    -- Transfer-specific fields
    to_user_id UUID,

    -- Scheduling details
    schedule_type VARCHAR(20) NOT NULL CHECK (schedule_type IN ('one-time', 'recurring')),
    execute_at TIMESTAMP WITH TIME ZONE NOT NULL,

    -- Recurring options
    recurrence_pattern VARCHAR(20) CHECK (recurrence_pattern IN ('daily', 'weekly', 'monthly', 'yearly')),
    recurrence_end_date TIMESTAMP WITH TIME ZONE,
    max_occurrences INTEGER CHECK (max_occurrences > 0),
    current_occurrence INTEGER NOT NULL DEFAULT 0,

    -- Status and control
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'cancelled', 'completed')),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,

    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_executed_at TIMESTAMP WITH TIME ZONE,
    next_execution_at TIMESTAMP WITH TIME ZONE,

    -- Foreign key constraints
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (to_user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Indexes for performance
CREATE INDEX idx_scheduled_transactions_user_id ON scheduled_transactions(user_id);
CREATE INDEX idx_scheduled_transactions_execute_at ON scheduled_transactions(execute_at);
CREATE INDEX idx_scheduled_transactions_status ON scheduled_transactions(status);
CREATE INDEX idx_scheduled_transactions_next_execution ON scheduled_transactions(next_execution_at);
CREATE INDEX idx_scheduled_transactions_active ON scheduled_transactions(is_active, status, execute_at);

-- Create execution_history table to track past executions
CREATE TABLE scheduled_transaction_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scheduled_transaction_id UUID NOT NULL,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(20) NOT NULL CHECK (status IN ('success', 'failed', 'skipped')),
    transaction_id UUID, -- Reference to the actual transaction if successful
    error_message TEXT,
    amount DECIMAL(15,2),
    currency VARCHAR(3),

    FOREIGN KEY (scheduled_transaction_id) REFERENCES scheduled_transactions(id) ON DELETE CASCADE,
    FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE SET NULL
);

-- Indexes for execution history
CREATE INDEX idx_executions_scheduled_id ON scheduled_transaction_executions(scheduled_transaction_id);
CREATE INDEX idx_executions_executed_at ON scheduled_transaction_executions(executed_at);
CREATE INDEX idx_executions_status ON scheduled_transaction_executions(status);
