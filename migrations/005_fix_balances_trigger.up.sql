-- Fix balances trigger to use correct column name
-- Drop the incorrect trigger first
DROP TRIGGER IF EXISTS update_balances_last_updated_at ON balances;

-- Create a specific trigger function for balances that uses last_updated_at
CREATE OR REPLACE FUNCTION update_balances_last_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create the correct trigger for balances table
CREATE TRIGGER update_balances_last_updated_at BEFORE UPDATE
    ON balances FOR EACH ROW EXECUTE FUNCTION update_balances_last_updated_at();
