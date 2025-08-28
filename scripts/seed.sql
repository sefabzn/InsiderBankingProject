-- Seed data for Go Banking Simulation
-- This script creates sample users and their initial balances

-- Insert test users (using bcrypt hash for password "password123")
INSERT INTO users (id, username, email, password_hash, role) VALUES 
    ('550e8400-e29b-41d4-a716-446655440001', 'kerem', 'kerem@example.com', '$2a$10$NSUEiKj7S5rACb22wwf2B.dhevBO0hI9CJgtYVSDCRRVHlb5oKoO2', 'user'),
    ('550e8400-e29b-41d4-a716-446655440002', 'sefa', 'sefa@example.com', '$2a$10$NSUEiKj7S5rACb22wwf2B.dhevBO0hI9CJgtYVSDCRRVHlb5oKoO2', 'user'),
    ('550e8400-e29b-41d4-a716-446655440003', 'admin', 'admin@example.com', '$2a$10$NSUEiKj7S5rACb22wwf2B.dhevBO0hI9CJgtYVSDCRRVHlb5oKoO2', 'admin')
ON CONFLICT (email) DO NOTHING;

-- Insert initial balances for the users
INSERT INTO balances (user_id, amount) VALUES 
    ('550e8400-e29b-41d4-a716-446655440001', 0.00),
    ('550e8400-e29b-41d4-a716-446655440002', 0.00),
    ('550e8400-e29b-41d4-a716-446655440003', 0.00)
ON CONFLICT (user_id) DO NOTHING;

-- Insert some sample audit logs
INSERT INTO audit_logs (entity_type, entity_id, action, details) VALUES 
    ('user', '550e8400-e29b-41d4-a716-446655440001', 'created', '{"initial_balance": 1000.00, "source": "seed"}'),
    ('user', '550e8400-e29b-41d4-a716-446655440002', 'created', '{"initial_balance": 500.00, "source": "seed"}'),
    ('user', '550e8400-e29b-41d4-a716-446655440003', 'created', '{"initial_balance": 0.00, "source": "seed", "role": "admin"}'),
    ('balance', '550e8400-e29b-41d4-a716-446655440001', 'initialized', '{"amount": 1000.00, "currency": "USD"}'),
    ('balance', '550e8400-e29b-41d4-a716-446655440002', 'initialized', '{"amount": 500.00, "currency": "USD"}'),
    ('balance', '550e8400-e29b-41d4-a716-446655440003', 'initialized', '{"amount": 0.00, "currency": "USD"}');

-- Display seeded data summary
SELECT 'Seeded users:' as info, count(*) as count FROM users;
SELECT 'Seeded balances:' as info, count(*) as count FROM balances;
SELECT 'Total balance in system:' as info, sum(amount) as total FROM balances;
