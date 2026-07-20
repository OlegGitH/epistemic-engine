BEGIN;

CREATE TABLE accounts (
    id text PRIMARY KEY,
    name text NOT NULL,
    slug text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE projects (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name text NOT NULL,
    slug text NOT NULL,
    repository text,
    owner text,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (account_id, slug)
);
CREATE INDEX projects_account_idx ON projects(account_id);

CREATE TABLE ai_systems (
    id text PRIMARY KEY,
    account_id text NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    project_id text NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    name text NOT NULL,
    provider text NOT NULL,
    model text NOT NULL,
    purpose text NOT NULL,
    data_classes jsonb NOT NULL DEFAULT '[]'::jsonb,
    tools jsonb NOT NULL DEFAULT '[]'::jsonb,
    owner text,
    status text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'retired')),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz
);
CREATE INDEX ai_systems_account_idx ON ai_systems(account_id);
CREATE INDEX ai_systems_project_idx ON ai_systems(project_id);

ALTER TABLE runs ADD COLUMN account_id text REFERENCES accounts(id) ON DELETE SET NULL;
ALTER TABLE runs ADD COLUMN project_id text REFERENCES projects(id) ON DELETE SET NULL;
ALTER TABLE runs ADD COLUMN ai_system_id text REFERENCES ai_systems(id) ON DELETE SET NULL;
CREATE INDEX runs_account_idx ON runs(account_id, created_at DESC);
CREATE INDEX runs_project_idx ON runs(project_id, created_at DESC);
CREATE INDEX runs_ai_system_idx ON runs(ai_system_id, created_at DESC);

COMMIT;
