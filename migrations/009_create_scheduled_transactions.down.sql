-- Drop scheduled transactions tables
DROP TABLE IF EXISTS scheduled_transaction_executions;
DROP INDEX IF EXISTS idx_scheduled_transactions_active;
DROP INDEX IF EXISTS idx_scheduled_transactions_next_execution;
DROP INDEX IF EXISTS idx_scheduled_transactions_status;
DROP INDEX IF EXISTS idx_scheduled_transactions_execute_at;
DROP INDEX IF EXISTS idx_scheduled_transactions_user_id;
DROP TABLE IF EXISTS scheduled_transactions;
