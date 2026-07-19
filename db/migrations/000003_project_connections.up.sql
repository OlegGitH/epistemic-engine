BEGIN;

CREATE TABLE project_connections (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    provider text NOT NULL CHECK (provider IN ('github-actions')),
    repository text NOT NULL,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'revoked')),
    token_hash char(64) NOT NULL UNIQUE,
    token_prefix text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    last_seen_at timestamptz
);
CREATE UNIQUE INDEX project_connections_active_project_idx ON project_connections(project_id) WHERE status = 'active';
CREATE INDEX project_connections_account_idx ON project_connections(account_id);

CREATE TABLE project_reports (
    id text PRIMARY KEY,
    external_id text NOT NULL,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    connection_id text NOT NULL REFERENCES project_connections(id) ON DELETE CASCADE,
    ai_system_id text REFERENCES ai_systems(id) ON DELETE SET NULL,
    tool text NOT NULL,
    status text NOT NULL CHECK (status IN ('passed', 'failed', 'error', 'pending', 'indeterminate')),
    exit_code integer NOT NULL,
    summary text NOT NULL,
    repository text,
    commit_sha text,
    branch text,
    workflow text,
    run_url text,
    details_json jsonb,
    received_at timestamptz NOT NULL,
    UNIQUE(connection_id, external_id)
);
CREATE INDEX project_reports_project_idx ON project_reports(project_id, received_at DESC);

CREATE TABLE published_certificates (
    id text PRIMARY KEY,
    external_id text NOT NULL,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    connection_id text NOT NULL REFERENCES project_connections(id) ON DELETE CASCADE,
    ai_system_id text REFERENCES ai_systems(id) ON DELETE SET NULL,
    decision_id text NOT NULL,
    run_id text,
    status text NOT NULL,
    action_allowed boolean NOT NULL,
    policy_version text NOT NULL,
    digest char(64) NOT NULL,
    artifact_hashes jsonb NOT NULL DEFAULT '[]'::jsonb,
    issued_at timestamptz NOT NULL,
    received_at timestamptz NOT NULL,
    certificate_json jsonb NOT NULL,
    UNIQUE(project_id, digest)
);
CREATE INDEX published_certificates_project_idx ON published_certificates(project_id, issued_at DESC);
CREATE INDEX published_certificates_ai_system_idx ON published_certificates(ai_system_id, issued_at DESC);

COMMIT;
