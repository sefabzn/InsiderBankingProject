-- Add currency support to balances and transactions tables
-- Add currency column to balances table
ALTER TABLE balances ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'USD';

-- Add currency column to transactions table
ALTER TABLE transactions ADD COLUMN currency VARCHAR(3) NOT NULL DEFAULT 'USD';

-- Add check constraints for valid currency codes (ISO 4217)
ALTER TABLE balances ADD CONSTRAINT chk_balances_currency CHECK (currency IN ('USD', 'EUR', 'GBP', 'JPY', 'CAD', 'AUD'));
ALTER TABLE transactions ADD CONSTRAINT chk_transactions_currency CHECK (currency IN ('USD', 'EUR', 'GBP', 'JPY', 'CAD', 'AUD'));

-- Create index on currency for better query performance
CREATE INDEX idx_balances_currency ON balances(currency);
CREATE INDEX idx_transactions_currency ON transactions(currency);

-- Update existing records to have USD currency (they were previously unspecified)
UPDATE balances SET currency = 'USD' WHERE currency = '';
UPDATE transactions SET currency = 'USD' WHERE currency = '';
