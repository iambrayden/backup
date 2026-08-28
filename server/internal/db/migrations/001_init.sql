-- +goose Up
-- +goose StatementBegin

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TYPE tenant_status AS ENUM ('active', 'suspended', 'trial');
CREATE TYPE device_status AS ENUM ('active', 'inactive', 'lost');
CREATE TYPE job_status AS ENUM ('pending', 'running', 'completed', 'failed', 'cancelled');
CREATE TYPE user_role AS ENUM ('vendor_admin', 'msp_technician', 'customer_admin', 'read_only');
CREATE TYPE os_type AS ENUM ('windows', 'linux', 'macos');

CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    status      tenant_status NOT NULL DEFAULT 'trial',
    dek_enc     BYTEA,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE resellers (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    slug        TEXT NOT NULL UNIQUE,
    tenant_id   UUID REFERENCES tenants(id) ON DELETE CASCADE,
    brand_config JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    role          user_role NOT NULL DEFAULT 'customer_admin',
    display_name  TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_users_tenant ON users(tenant_id);
CREATE INDEX idx_users_email  ON users(email);

CREATE TABLE sites (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_sites_tenant ON sites(tenant_id);

CREATE TABLE devices (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    site_id       UUID REFERENCES sites(id) ON DELETE SET NULL,
    hostname      TEXT NOT NULL,
    os_type       os_type NOT NULL,
    arch          TEXT NOT NULL,
    agent_version TEXT NOT NULL DEFAULT '',
    status        device_status NOT NULL DEFAULT 'active',
    last_seen_at  TIMESTAMPTZ,
    enrolled_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_devices_tenant  ON devices(tenant_id);
CREATE INDEX idx_devices_site    ON devices(site_id);
CREATE INDEX idx_devices_status  ON devices(status);

CREATE TABLE agent_enrollments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    token_hash   TEXT NOT NULL UNIQUE,
    device_id    UUID REFERENCES devices(id) ON DELETE SET NULL,
    label        TEXT NOT NULL DEFAULT '',
    used_at      TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_enrollments_token  ON agent_enrollments(token_hash);
CREATE INDEX idx_enrollments_tenant ON agent_enrollments(tenant_id);

CREATE TABLE device_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    token_hash  TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_device_tokens_hash ON device_tokens(token_hash);

CREATE TABLE storage_targets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    label           TEXT NOT NULL DEFAULT 'primary',
    provider        TEXT NOT NULL,
    endpoint        TEXT NOT NULL DEFAULT '',
    bucket          TEXT NOT NULL,
    region          TEXT NOT NULL DEFAULT '',
    credentials_enc BYTEA NOT NULL,
    is_primary      BOOLEAN NOT NULL DEFAULT TRUE,
    path_style      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_storage_targets_tenant ON storage_targets(tenant_id);

CREATE TABLE backup_jobs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id         UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    status            job_status NOT NULL DEFAULT 'pending',
    bytes_scanned     BIGINT NOT NULL DEFAULT 0,
    bytes_changed     BIGINT NOT NULL DEFAULT 0,
    chunks_total      BIGINT NOT NULL DEFAULT 0,
    chunks_uploaded   BIGINT NOT NULL DEFAULT 0,
    chunks_deduped    BIGINT NOT NULL DEFAULT 0,
    error_msg         TEXT,
    started_at        TIMESTAMPTZ,
    finished_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_jobs_device  ON backup_jobs(device_id);
CREATE INDEX idx_jobs_tenant  ON backup_jobs(tenant_id);
CREATE INDEX idx_jobs_status  ON backup_jobs(status);

CREATE TABLE snapshots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    tenant_id       UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    job_id          UUID REFERENCES backup_jobs(id) ON DELETE SET NULL,
    file_count      BIGINT NOT NULL DEFAULT 0,
    total_size      BIGINT NOT NULL DEFAULT 0,
    dedupe_size     BIGINT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_snapshots_device    ON snapshots(device_id);
CREATE INDEX idx_snapshots_tenant    ON snapshots(tenant_id);
CREATE INDEX idx_snapshots_created   ON snapshots(created_at DESC);

CREATE TABLE chunks (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    hash               TEXT NOT NULL,
    plaintext_size     BIGINT NOT NULL,
    stored_size        BIGINT NOT NULL,
    storage_target_id  UUID NOT NULL REFERENCES storage_targets(id) ON DELETE RESTRICT,
    object_key         TEXT NOT NULL,
    ref_count          BIGINT NOT NULL DEFAULT 1,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, hash)
);
CREATE INDEX idx_chunks_tenant_hash ON chunks(tenant_id, hash);

CREATE TABLE file_tree_entries (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id  UUID NOT NULL REFERENCES snapshots(id) ON DELETE CASCADE,
    path         TEXT NOT NULL,
    parent_path  TEXT NOT NULL,
    name         TEXT NOT NULL,
    size         BIGINT NOT NULL DEFAULT 0,
    mtime        TIMESTAMPTZ,
    mode         INTEGER NOT NULL DEFAULT 0,
    is_dir       BOOLEAN NOT NULL DEFAULT FALSE,
    chunk_hashes TEXT[] NOT NULL DEFAULT '{}',
    UNIQUE(snapshot_id, path)
);
CREATE INDEX idx_file_tree_snapshot_parent ON file_tree_entries(snapshot_id, parent_path);
CREATE INDEX idx_file_tree_snapshot_path   ON file_tree_entries(snapshot_id, path text_pattern_ops);

CREATE TABLE replication_jobs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    source_target_id  UUID NOT NULL REFERENCES storage_targets(id),
    dest_target_id    UUID NOT NULL REFERENCES storage_targets(id),
    status            job_status NOT NULL DEFAULT 'pending',
    chunks_copied     BIGINT NOT NULL DEFAULT 0,
    bytes_copied      BIGINT NOT NULL DEFAULT 0,
    error_msg         TEXT,
    started_at        TIMESTAMPTZ,
    finished_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE retention_policies (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    device_id    UUID REFERENCES devices(id) ON DELETE CASCADE,
    daily_keep   INT NOT NULL DEFAULT 7,
    weekly_keep  INT NOT NULL DEFAULT 4,
    monthly_keep INT NOT NULL DEFAULT 12,
    yearly_keep  INT NOT NULL DEFAULT 3,
    legal_hold   BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE audit_log (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID REFERENCES tenants(id) ON DELETE SET NULL,
    user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   UUID,
    meta          JSONB NOT NULL DEFAULT '{}',
    ip_address    TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_audit_tenant  ON audit_log(tenant_id, created_at DESC);
CREATE INDEX idx_audit_user    ON audit_log(user_id, created_at DESC);

CREATE TABLE usage_records (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    period_date  DATE NOT NULL,
    device_count INT NOT NULL DEFAULT 0,
    storage_gb   NUMERIC(12,3) NOT NULL DEFAULT 0,
    stripe_usage_recorded BOOLEAN NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, period_date)
);

CREATE TABLE alert_rules (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    event_type   TEXT NOT NULL,
    threshold    JSONB NOT NULL DEFAULT '{}',
    channels     JSONB NOT NULL DEFAULT '[]',
    enabled      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE alert_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    rule_id      UUID REFERENCES alert_rules(id) ON DELETE SET NULL,
    device_id    UUID REFERENCES devices(id) ON DELETE SET NULL,
    event_type   TEXT NOT NULL,
    severity     TEXT NOT NULL DEFAULT 'warning',
    message      TEXT NOT NULL,
    resolved_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_alert_events_tenant ON alert_events(tenant_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS alert_events CASCADE;
DROP TABLE IF EXISTS alert_rules CASCADE;
DROP TABLE IF EXISTS usage_records CASCADE;
DROP TABLE IF EXISTS audit_log CASCADE;
DROP TABLE IF EXISTS retention_policies CASCADE;
DROP TABLE IF EXISTS replication_jobs CASCADE;
DROP TABLE IF EXISTS file_tree_entries CASCADE;
DROP TABLE IF EXISTS chunks CASCADE;
DROP TABLE IF EXISTS snapshots CASCADE;
DROP TABLE IF EXISTS backup_jobs CASCADE;
DROP TABLE IF EXISTS storage_targets CASCADE;
DROP TABLE IF EXISTS device_tokens CASCADE;
DROP TABLE IF EXISTS agent_enrollments CASCADE;
DROP TABLE IF EXISTS devices CASCADE;
DROP TABLE IF EXISTS sites CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS resellers CASCADE;
DROP TABLE IF EXISTS tenants CASCADE;
DROP TYPE IF EXISTS alert_severity;
DROP TYPE IF EXISTS os_type;
DROP TYPE IF EXISTS user_role;
DROP TYPE IF EXISTS job_status;
DROP TYPE IF EXISTS device_status;
DROP TYPE IF EXISTS tenant_status;
DROP EXTENSION IF EXISTS "pgcrypto";
-- +goose StatementEnd
