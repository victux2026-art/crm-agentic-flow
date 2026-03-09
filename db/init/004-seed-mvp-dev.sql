-- Development-only seed data for the MVP schema.
-- Default login:
-- email: admin@crmflow.local
-- password: admin123
-- tenant_slug: demo

INSERT INTO tenants (id, name, slug, plan, status)
VALUES (1, 'Demo Tenant', 'demo', 'starter', 'active')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, email, password_hash, full_name, status)
VALUES (
    1,
    'admin@crmflow.local',
    '$2a$10$Iag1qEZ08E5uPWumhniku.QxLzn3Far1wFXKLFY/GrM2o3nRKPWFa',
    'Demo Admin',
    'active'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO tenant_memberships (tenant_id, user_id, role, status)
VALUES (1, 1, 'owner', 'active')
ON CONFLICT (tenant_id, user_id) DO NOTHING;
