// Package epistemic defines the stable, vendor-neutral Epistemic Protocol API.
// It intentionally has no dependency on an engine, database, UI, model vendor,
// or telemetry framework.
package epistemic

import (
	"encoding/json"
	"time"
)

const Version = "0.1"

type Source struct {
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Instance string `json:"instance,omitempty"`
}

type Subject struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Context struct {
	DecisionID  string `json:"decision_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	Correlation string `json:"correlation_id,omitempty"`
	ParentID    string `json:"parent_id,omitempty"`
}

type Ordering struct {
	Sequence  int64  `json:"sequence,omitempty"`
	Partition string `json:"partition,omitempty"`
}

type Event struct {
	SpecVersion    string                     `json:"spec_version"`
	ID             string                     `json:"id"`
	Type           string                     `json:"type"`
	Source         Source                     `json:"source"`
	Subject        Subject                    `json:"subject"`
	Time           time.Time                  `json:"time"`
	Context        Context                    `json:"context,omitempty"`
	Ordering       Ordering                   `json:"ordering,omitempty"`
	IdempotencyKey string                     `json:"idempotency_key,omitempty"`
	Data           json.RawMessage            `json:"data"`
	Extensions     map[string]json.RawMessage `json:"extensions,omitempty"`
	Unknown        map[string]json.RawMessage `json:"-"`
}

type Action struct {
	Type      string  `json:"type"`
	Subject   Subject `json:"subject"`
	RiskLevel string  `json:"risk_level,omitempty"`
}

type Requirement struct {
	ID            string   `json:"id"`
	Description   string   `json:"description"`
	Critical      bool     `json:"critical"`
	EvidenceTypes []string `json:"evidence_types,omitempty"`
}

type Approval struct {
	Approved bool   `json:"approved"`
	Actor    string `json:"actor,omitempty"`
	Time     string `json:"time,omitempty"`
}

type DecisionRequest struct {
	SpecVersion    string                     `json:"spec_version"`
	DecisionID     string                     `json:"decision_id,omitempty"`
	Recommendation string                     `json:"recommendation"`
	Action         Action                     `json:"action"`
	Context        Context                    `json:"context,omitempty"`
	Mode           string                     `json:"mode,omitempty"`
	Requirements   []Requirement              `json:"requirements,omitempty"`
	Events         []Event                    `json:"events,omitempty"`
	Approval       Approval                   `json:"approval,omitempty"`
	Metadata       map[string]json.RawMessage `json:"metadata,omitempty"`
}

type Reason struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	SubjectID string `json:"subject_id,omitempty"`
}

type DecisionResult struct {
	SpecVersion   string       `json:"spec_version"`
	DecisionID    string       `json:"decision_id"`
	Status        string       `json:"status"`
	ActionAllowed bool         `json:"action_allowed"`
	Reasons       []Reason     `json:"reasons"`
	Conditions    []string     `json:"conditions"`
	EvaluatedAt   time.Time    `json:"evaluated_at"`
	Context       Context      `json:"context,omitempty"`
	Certificate   *Certificate `json:"certificate,omitempty"`
}

type Proof struct {
	Algorithm string `json:"algorithm"`
	Digest    string `json:"digest"`
}

type Certificate struct {
	SpecVersion    string          `json:"spec_version"`
	ID             string          `json:"id"`
	DecisionID     string          `json:"decision_id"`
	Result         json.RawMessage `json:"result"`
	ArtifactHashes []string        `json:"artifact_hashes"`
	IssuedAt       time.Time       `json:"issued_at"`
	Supersedes     string          `json:"supersedes,omitempty"`
	Proof          Proof           `json:"proof"`
}

type Error struct {
	SpecVersion string         `json:"spec_version"`
	Code        string         `json:"code"`
	Message     string         `json:"message"`
	Retryable   bool           `json:"retryable"`
	Details     map[string]any `json:"details,omitempty"`
}

type Capabilities struct {
	ProtocolVersions []string `json:"protocol_versions"`
	Transports       []string `json:"transports"`
	EventTypes       []string `json:"event_types"`
	DecisionStatuses []string `json:"decision_statuses"`
	Features         []string `json:"features"`
	Limits           Limits   `json:"limits"`
}

type Limits struct {
	MaxEventBytes int `json:"max_event_bytes"`
	MaxBatchSize  int `json:"max_batch_size"`
}
