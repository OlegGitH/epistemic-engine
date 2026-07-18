BEGIN;

CREATE TABLE runs (
    id text PRIMARY KEY,
    external_trace_id text UNIQUE,
    title text NOT NULL,
    goal text NOT NULL,
    source text NOT NULL,
    recommendation text NOT NULL,
    status text NOT NULL CHECK (status IN ('ingesting', 'analyzed')),
    decision_id text UNIQUE,
    raw jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE decisions (
    id text PRIMARY KEY,
    run_id text NOT NULL UNIQUE REFERENCES runs(id) ON DELETE CASCADE,
    recommendation text NOT NULL,
    action_type text NOT NULL,
    subject text NOT NULL,
    risk_level text NOT NULL CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
    policy_version text NOT NULL,
    verdict text CHECK (verdict IN ('VERIFIED', 'VERIFIED_WITH_CONDITIONS', 'INSUFFICIENT_EVIDENCE', 'CONTRADICTED', 'REJECTED')),
    action_allowed boolean NOT NULL DEFAULT false,
    human_approved boolean NOT NULL DEFAULT false,
    conditions jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at timestamptz NOT NULL DEFAULT now(),
    evaluated_at timestamptz
);

ALTER TABLE runs ADD CONSTRAINT runs_decision_fk FOREIGN KEY (decision_id) REFERENCES decisions(id) DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE run_events (
    id text PRIMARY KEY,
    external_id text,
    run_id text NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    sequence bigint NOT NULL,
    event_type text NOT NULL,
    source text NOT NULL,
    correlation_id text,
    occurred_at timestamptz NOT NULL,
    payload_json jsonb NOT NULL,
    payload_sha256 char(64) NOT NULL,
    UNIQUE (run_id, sequence)
);
CREATE UNIQUE INDEX run_events_external_id_idx ON run_events (run_id, external_id) WHERE external_id IS NOT NULL AND external_id <> '';

CREATE TABLE claims (
    id text PRIMARY KEY,
    decision_id text NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
    statement text NOT NULL,
    importance text NOT NULL CHECK (importance IN ('critical', 'supporting')),
    critical boolean NOT NULL,
    scope_json jsonb NOT NULL,
    required_evidence_types jsonb NOT NULL DEFAULT '[]'::jsonb,
    state text NOT NULL CHECK (state IN ('supported', 'partially_supported', 'unsupported', 'contradicted', 'stale', 'verification_pending', 'externally_verified', 'rejected')),
    justification text NOT NULL,
    support_score jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE evidence (
    id text PRIMARY KEY,
    run_id text NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    evidence_type text NOT NULL,
    source text NOT NULL,
    summary text NOT NULL,
    uri text,
    content_hash char(64) NOT NULL,
    freshness_at timestamptz NOT NULL,
    metadata_json jsonb,
    raw jsonb
);

CREATE TABLE claim_evidence (
    claim_id text NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
    evidence_id text NOT NULL REFERENCES evidence(id) ON DELETE CASCADE,
    relation text NOT NULL CHECK (relation IN ('supports', 'contradicts', 'qualifies', 'supersedes', 'derived_from')),
    strength double precision NOT NULL DEFAULT 1 CHECK (strength >= 0 AND strength <= 1),
    rationale text NOT NULL,
    PRIMARY KEY (claim_id, evidence_id, relation)
);

CREATE TABLE relations (
    id text PRIMARY KEY,
    from_id text NOT NULL,
    to_id text NOT NULL,
    relation_type text NOT NULL CHECK (relation_type IN ('supports', 'contradicts', 'qualifies', 'supersedes', 'derived_from', 'verified_by')),
    rationale text NOT NULL,
    UNIQUE (from_id, to_id, relation_type)
);
CREATE INDEX relations_from_idx ON relations (from_id);
CREATE INDEX relations_to_idx ON relations (to_id);

CREATE TABLE assumptions (
    id text PRIMARY KEY,
    decision_id text NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
    claim_id text REFERENCES claims(id) ON DELETE CASCADE,
    statement text NOT NULL,
    critical boolean NOT NULL,
    status text NOT NULL,
    impact text NOT NULL
);

CREATE TABLE unknowns (
    id text PRIMARY KEY,
    decision_id text NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
    question text NOT NULL,
    critical boolean NOT NULL,
    resolved boolean NOT NULL DEFAULT false
);

CREATE TABLE verifications (
    id text PRIMARY KEY,
    decision_id text NOT NULL REFERENCES decisions(id) ON DELETE CASCADE,
    claim_id text NOT NULL REFERENCES claims(id) ON DELETE CASCADE,
    kind text NOT NULL CHECK (kind IN ('test', 'human')),
    check_description text NOT NULL,
    specification_json jsonb NOT NULL,
    environment text NOT NULL CHECK (environment IN ('sandbox', 'staging')),
    status text NOT NULL CHECK (status IN ('planned', 'running', 'completed')),
    requires_approval boolean NOT NULL DEFAULT true,
    approved boolean NOT NULL DEFAULT false,
    approved_by text,
    outcome text CHECK (outcome IN ('passed', 'failed')),
    result_json jsonb,
    artifact_hash char(64),
    created_at timestamptz NOT NULL DEFAULT now(),
    executed_at timestamptz
);

CREATE TABLE policies (
    id text PRIMARY KEY,
    version text NOT NULL UNIQUE,
    definition_json jsonb NOT NULL,
    active boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO policies (id, version, definition_json, active) VALUES (
    'policy_deployment_readiness_v1',
    'deployment-readiness/v1',
    '{"critical_unsupported":"block","critical_contradicted":"block","human_approval_required":true,"production_execution":false}'::jsonb,
    true
);

CREATE TABLE proofs (
    id text PRIMARY KEY,
    decision_id text NOT NULL UNIQUE REFERENCES decisions(id) ON DELETE RESTRICT,
    certificate_json jsonb NOT NULL,
    certificate_hash char(64) NOT NULL UNIQUE,
    created_at timestamptz NOT NULL
);

COMMIT;
