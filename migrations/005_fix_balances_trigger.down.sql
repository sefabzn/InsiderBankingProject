-- Rollback balances trigger fix
DROP TRIGGER IF EXISTS update_balances_last_updated_at ON balances;
DROP FUNCTION IF EXISTS update_balances_last_updated_at();

-- Recreate the original (incorrect) trigger that was in the balances migration
CREATE TRIGGER update_balances_last_updated_at BEFORE UPDATE
    ON balances FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
