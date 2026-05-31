INSERT INTO clients (id, name, created_at)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'Demo Client', now())
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, email, password_hash, role, client_id, created_at)
VALUES
    (
        '22222222-2222-2222-2222-222222222222',
        'marketer@example.com',
        '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
        'marketer',
        NULL,
        now()
    ),
    (
        '33333333-3333-3333-3333-333333333333',
        'client@example.com',
        '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy',
        'client',
        '11111111-1111-1111-1111-111111111111',
        now()
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO links (
    id,
    short_code,
    original_url,
    created_by,
    campaign_name,
    client_id,
    expires_at,
    is_active,
    created_at
)
VALUES (
    '44444444-4444-4444-4444-444444444444',
    'xK9mP',
    'https://example.com',
    '22222222-2222-2222-2222-222222222222',
    'Demo Campaign',
    '11111111-1111-1111-1111-111111111111',
    now() + interval '30 days',
    true,
    now()
)
ON CONFLICT (id) DO NOTHING;
