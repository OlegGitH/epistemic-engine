package domain

import (
	"encoding/json"
	"time"
)

type ProjectConnection struct {
	ID          string     `json:"id"`
	AccountID   string     `json:"account_id"`
	ProjectID   string     `json:"project_id"`
	Provider    string     `json:"provider"`
	Repository  string     `json:"repository"`
	Status      string     `json:"status"`
	TokenHash   string     `json:"-"`
	TokenPrefix string     `json:"token_prefix"`
	CreatedAt   time.Time  `json:"created_at"`
	LastSeenAt  *time.Time `json:"last_seen_at,omitempty"`
}

type ProjectReport struct {
	ID           string          `json:"id"`
	ExternalID   string          `json:"external_id"`
	AccountID    string          `json:"account_id"`
	ProjectID    string          `json:"project_id"`
	ConnectionID string          `json:"connection_id"`
	AISystemID   string          `json:"ai_system_id,omitempty"`
	Tool         string          `json:"tool"`
	Status       string          `json:"status"`
	ExitCode     int             `json:"exit_code"`
	Summary      string          `json:"summary"`
	Repository   string          `json:"repository,omitempty"`
	CommitSHA    string          `json:"commit_sha,omitempty"`
	Branch       string          `json:"branch,omitempty"`
	Workflow     string          `json:"workflow,omitempty"`
	RunURL       string          `json:"run_url,omitempty"`
	Details      json.RawMessage `json:"details,omitempty"`
	ReceivedAt   time.Time       `json:"received_at"`
}

type PublishedCertificate struct {
	ID             string          `json:"id"`
	ExternalID     string          `json:"external_id"`
	AccountID      string          `json:"account_id"`
	ProjectID      string          `json:"project_id"`
	ConnectionID   string          `json:"connection_id"`
	AISystemID     string          `json:"ai_system_id,omitempty"`
	DecisionID     string          `json:"decision_id"`
	RunID          string          `json:"run_id,omitempty"`
	Status         string          `json:"status"`
	ActionAllowed  bool            `json:"action_allowed"`
	PolicyVersion  string          `json:"policy_version"`
	Digest         string          `json:"digest"`
	ArtifactHashes []string        `json:"artifact_hashes"`
	IssuedAt       time.Time       `json:"issued_at"`
	ReceivedAt     time.Time       `json:"received_at"`
	Raw            json.RawMessage `json:"raw,omitempty"`
}

type ProjectIngest struct {
	Connection  ProjectConnection
	Report      *ProjectReport
	Certificate *PublishedCertificate
}

type ConnectionSummary struct {
	ProjectConnection
	ProjectName string `json:"project_name"`
	Reports     int    `json:"reports"`
}
