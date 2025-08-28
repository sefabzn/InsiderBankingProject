-- Create balances table
CREATE TABLE IF NOT EXISTS balances (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    amount NUMERIC(18,2) NOT NULL DEFAULT 0.00,
    last_updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add constraint to ensure amount is not negative
ALTER TABLE balances ADD CONSTRAINT chk_balances_amount_non_negative CHECK (amount >= 0);

-- Create index for faster lookups
CREATE INDEX IF NOT EXISTS idx_balances_user_id ON balances(user_id);
CREATE INDEX IF NOT EXISTS idx_balances_last_updated ON balances(last_updated_at);

-- Create trigger to update last_updated_at timestamp
CREATE TRIGGER update_balances_last_updated_at BEFORE UPDATE
    ON balances FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
