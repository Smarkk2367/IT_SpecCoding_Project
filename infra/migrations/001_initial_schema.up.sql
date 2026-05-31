CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE clients (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    name text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL UNIQUE,
    password_hash text NOT NULL,
    role text NOT NULL CHECK (role IN ('marketer', 'client')),
    client_id uuid NULL REFERENCES clients(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    CHECK (
        (role = 'client' AND client_id IS NOT NULL)
        OR (role = 'marketer' AND client_id IS NULL)
    )
);

CREATE TABLE links (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    short_code text NOT NULL UNIQUE,
    original_url text NOT NULL,
    created_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    campaign_name text NULL,
    client_id uuid NULL REFERENCES clients(id) ON DELETE RESTRICT,
    expires_at timestamptz NULL,
    is_active boolean NOT NULL DEFAULT true,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz NULL,
    CHECK (expires_at IS NULL OR expires_at <= created_at + interval '365 days')
);

CREATE TABLE clicks (
    id bigserial PRIMARY KEY,
    link_id uuid NOT NULL REFERENCES links(id) ON DELETE RESTRICT,
    clicked_at timestamptz NOT NULL,
    country text NULL,
    city text NULL,
    device_type text NULL CHECK (device_type IS NULL OR device_type IN ('mobile', 'desktop', 'tablet')),
    browser text NULL,
    os text NULL,
    referrer text NULL,
    ip_hash text NULL,
    event_id uuid NOT NULL UNIQUE,
    user_agent text NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE reports (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    status text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'done', 'failed')),
    requested_by uuid NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    client_id uuid NULL REFERENCES clients(id) ON DELETE RESTRICT,
    date_from timestamptz NOT NULL,
    date_to timestamptz NOT NULL,
    file_path text NULL,
    error_message text NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    completed_at timestamptz NULL,
    CHECK (date_from < date_to),
    CHECK (
        (status = 'done' AND file_path IS NOT NULL)
        OR (status <> 'done')
    )
);

CREATE TABLE report_links (
    report_id uuid NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    link_id uuid NOT NULL REFERENCES links(id) ON DELETE RESTRICT,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (report_id, link_id)
);

CREATE TABLE failed_events (
    id bigserial PRIMARY KEY,
    event_id uuid NOT NULL UNIQUE,
    event_type text NOT NULL,
    stream text NOT NULL,
    payload jsonb NOT NULL,
    error_message text NOT NULL,
    failed_at timestamptz NOT NULL DEFAULT now(),
    reprocessed_at timestamptz NULL
);

CREATE INDEX idx_users_client_id ON users(client_id);

CREATE INDEX idx_links_client_campaign ON links(client_id, campaign_name);
CREATE INDEX idx_links_expires_at ON links(expires_at) WHERE expires_at IS NOT NULL;
CREATE INDEX idx_links_active_short_code ON links(short_code) WHERE is_active = true AND deleted_at IS NULL;

CREATE INDEX idx_clicks_link_clicked_at ON clicks(link_id, clicked_at);
CREATE INDEX idx_clicks_country ON clicks(country) WHERE country IS NOT NULL;
CREATE INDEX idx_clicks_device_type ON clicks(device_type) WHERE device_type IS NOT NULL;
CREATE INDEX idx_clicks_referrer ON clicks(referrer) WHERE referrer IS NOT NULL;
CREATE INDEX idx_clicks_link_ip_hash ON clicks(link_id, ip_hash) WHERE ip_hash IS NOT NULL;

CREATE INDEX idx_reports_status_created_at ON reports(status, created_at);
CREATE INDEX idx_reports_requested_by_created_at ON reports(requested_by, created_at);
CREATE INDEX idx_report_links_link_id ON report_links(link_id);

CREATE INDEX idx_failed_events_stream_failed_at ON failed_events(stream, failed_at);
