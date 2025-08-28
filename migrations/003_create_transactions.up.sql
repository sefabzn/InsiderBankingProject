-- Create transactions table
CREATE TABLE IF NOT EXISTS transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    from_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    to_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    amount NUMERIC(18,2) NOT NULL,
    type VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add constraints for valid transaction types
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_type 
    CHECK (type IN ('credit', 'debit', 'transfer'));

-- Add constraints for valid transaction status
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_status 
    CHECK (status IN ('pending', 'success', 'failed'));

-- Add constraint to ensure amount is positive
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_amount_positive 
    CHECK (amount >= 0);

-- Add constraint for transfer transactions (must have both from and to users)
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_transfer_users
    CHECK (
        (type = 'transfer' AND from_user_id IS NOT NULL AND to_user_id IS NOT NULL) OR
        (type != 'transfer')
    );

-- Add constraint for credit transactions (must have to_user_id, no from_user_id)
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_credit_users
    CHECK (
        (type = 'credit' AND to_user_id IS NOT NULL AND from_user_id IS NULL) OR
        (type != 'credit')
    );

-- Add constraint for debit transactions (must have from_user_id, no to_user_id)
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_debit_users
    CHECK (
        (type = 'debit' AND from_user_id IS NOT NULL AND to_user_id IS NULL) OR
        (type != 'debit')
    );

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_transactions_from_user ON transactions(from_user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_to_user ON transactions(to_user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_type ON transactions(type);
CREATE INDEX IF NOT EXISTS idx_transactions_status ON transactions(status);
CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at);

-- Composite index for user transaction history queries
CREATE INDEX IF NOT EXISTS idx_transactions_user_history ON transactions(from_user_id, to_user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transactions_type_status ON transactions(type, status);
