-- Drop trigger
DROP TRIGGER IF EXISTS update_balances_last_updated_at ON balances;

-- Drop indexes
DROP INDEX IF EXISTS idx_balances_user_id;
DROP INDEX IF EXISTS idx_balances_last_updated;

-- Drop balances table
DROP TABLE IF EXISTS balances;
