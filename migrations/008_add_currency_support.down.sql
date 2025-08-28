-- Remove currency support from balances and transactions tables
-- Drop indexes
DROP INDEX IF EXISTS idx_transactions_currency;
DROP INDEX IF EXISTS idx_balances_currency;

-- Drop check constraints
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS chk_transactions_currency;
ALTER TABLE balances DROP CONSTRAINT IF EXISTS chk_balances_currency;

-- Drop currency columns
ALTER TABLE transactions DROP COLUMN IF EXISTS currency;
ALTER TABLE balances DROP COLUMN IF EXISTS currency;
