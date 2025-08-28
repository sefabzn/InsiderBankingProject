-- Drop indexes
DROP INDEX IF EXISTS idx_transactions_from_user;
DROP INDEX IF EXISTS idx_transactions_to_user;
DROP INDEX IF EXISTS idx_transactions_type;
DROP INDEX IF EXISTS idx_transactions_status;
DROP INDEX IF EXISTS idx_transactions_created_at;
DROP INDEX IF EXISTS idx_transactions_user_history;
DROP INDEX IF EXISTS idx_transactions_type_status;

-- Drop transactions table
DROP TABLE IF EXISTS transactions;
