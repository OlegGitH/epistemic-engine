package domain

import (
	"encoding/json"
	"time"
)

type RunStatus string

const (
	RunIngesting RunStatus = "ingesting"
	RunAnalyzed  RunStatus = "analyzed"
)

type ClaimState string

const (
	ClaimSupported           ClaimState = "supported"
	ClaimPartiallySupported  ClaimState = "partially_supported"
	ClaimUnsupported         ClaimState = "unsupported"
	ClaimContradicted        ClaimState = "contradicted"
	ClaimStale               ClaimState = "stale"
	ClaimVerificationPending ClaimState = "verification_pending"
	ClaimExternallyVerified  ClaimState = "externally_verified"
	ClaimRejected            ClaimState = "rejected"
)

type RelationType string

const (
	RelationSupports    RelationType = "supports"
	RelationContradicts RelationType = "contradicts"
	RelationQualifies   RelationType = "qualifies"
	RelationSupersedes  RelationType = "supersedes"
	RelationDerivedFrom RelationType = "derived_from"
	RelationVerifiedBy  RelationType = "verified_by"
)

type Verdict string

const (
	VerdictVerified               Verdict = "VERIFIED"
	VerdictVerifiedWithConditions Verdict = "VERIFIED_WITH_CONDITIONS"
	VerdictInsufficientEvidence   Verdict = "INSUFFICIENT_EVIDENCE"
	VerdictContradicted           Verdict = "CONTRADICTED"
	VerdictRejected               Verdict = "REJECTED"
)

type Run struct {
	ID              string          `json:"id"`
	AccountID       string          `json:"account_id,omitempty"`
	ProjectID       string          `json:"project_id,omitempty"`
	AISystemID      string          `json:"ai_system_id,omitempty"`
	ExternalTraceID string          `json:"external_trace_id,omitempty"`
	Title           string          `json:"title"`
	Goal            string          `json:"goal,omitempty"`
	Source          string          `json:"source"`
	Recommendation  string          `json:"recommendation"`
	Status          RunStatus       `json:"status"`
	DecisionID      string          `json:"decision_id"`
	Events          []Event         `json:"events"`
	Raw             json.RawMessage `json:"raw,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

type Event struct {
	ID            string          `json:"id"`
	ExternalID    string          `json:"external_id,omitempty"`
	Sequence      int64           `json:"sequence"`
	Type          string          `json:"type"`
	Source        string          `json:"source"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Payload       json.RawMessage `json:"payload"`
}

type Decision struct {
	ID             string     `json:"id"`
	RunID          string     `json:"run_id"`
	Recommendation string     `json:"recommendation"`
	ActionType     string     `json:"action_type"`
	Subject        string     `json:"subject"`
	RiskLevel      string     `json:"risk_level"`
	PolicyVersion  string     `json:"policy_version"`
	Verdict        Verdict    `json:"verdict,omitempty"`
	ActionAllowed  bool       `json:"action_allowed"`
	HumanApproved  bool       `json:"human_approved"`
	Conditions     []string   `json:"conditions"`
	ClaimIDs       []string   `json:"claim_ids"`
	CreatedAt      time.Time  `json:"created_at"`
	EvaluatedAt    *time.Time `json:"evaluated_at,omitempty"`
}

type SupportScore struct {
	Value                      float64 `json:"value"`
	EvidenceCoverage           float64 `json:"evidence_coverage"`
	SourceQuality              float64 `json:"source_quality"`
	SourceIndependence         float64 `json:"source_independence"`
	Freshness                  float64 `json:"freshness"`
	ScopeMatch                 float64 `json:"scope_match"`
	DirectVerificationStrength float64 `json:"direct_verification_strength"`
	ContradictionBurden        float64 `json:"contradiction_burden"`
	Semantics                  string  `json:"semantics"`
}

type Claim struct {
	ID                    string       `json:"id"`
	DecisionID            string       `json:"decision_id"`
	Statement             string       `json:"statement"`
	Scope                 string       `json:"scope"`
	Critical              bool         `json:"critical"`
	Importance            string       `json:"importance"`
	RequiredEvidenceTypes []string     `json:"required_evidence_types"`
	State                 ClaimState   `json:"state"`
	Justification         string       `json:"justification"`
	Support               SupportScore `json:"support"`
	EvidenceIDs           []string     `json:"evidence_ids"`
	CreatedAt             time.Time    `json:"created_at"`
}

type Evidence struct {
	ID          string          `json:"id"`
	RunID       string          `json:"run_id"`
	Kind        string          `json:"kind"`
	Source      string          `json:"source"`
	Summary     string          `json:"summary"`
	URI         string          `json:"uri,omitempty"`
	ContentHash string          `json:"content_hash"`
	ObservedAt  time.Time       `json:"observed_at"`
	Raw         json.RawMessage `json:"raw,omitempty"`
	Metadata    json.RawMessage `json:"metadata,omitempty"`
}

type Relation struct {
	ID        string       `json:"id"`
	FromID    string       `json:"from_id"`
	ToID      string       `json:"to_id"`
	Type      RelationType `json:"type"`
	Rationale string       `json:"rationale"`
}

type Assumption struct {
	ID         string `json:"id"`
	DecisionID string `json:"decision_id"`
	ClaimID    string `json:"claim_id,omitempty"`
	Statement  string `json:"statement"`
	Critical   bool   `json:"critical"`
	Status     string `json:"status"`
	Impact     string `json:"impact"`
}

type Unknown struct {
	ID         string `json:"id"`
	DecisionID string `json:"decision_id"`
	Question   string `json:"question"`
	Critical   bool   `json:"critical"`
	Resolved   bool   `json:"resolved"`
}

type Verification struct {
	ID               string          `json:"id"`
	DecisionID       string          `json:"decision_id"`
	ClaimID          string          `json:"claim_id"`
	Check            string          `json:"check"`
	Kind             string          `json:"kind"`
	Specification    json.RawMessage `json:"specification"`
	Environment      string          `json:"environment"`
	Status           string          `json:"status"`
	RequiresApproval bool            `json:"requires_approval"`
	Approved         bool            `json:"approved"`
	ApprovedBy       string          `json:"approved_by,omitempty"`
	Outcome          string          `json:"outcome,omitempty"`
	Artifact         json.RawMessage `json:"artifact,omitempty"`
	ArtifactHash     string          `json:"artifact_hash,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	ExecutedAt       *time.Time      `json:"executed_at,omitempty"`
}

type Graph struct {
	Run           Run            `json:"run"`
	Decision      Decision       `json:"decision"`
	Claims        []Claim        `json:"claims"`
	Evidence      []Evidence     `json:"evidence"`
	Relations     []Relation     `json:"relations"`
	Assumptions   []Assumption   `json:"assumptions"`
	Unknowns      []Unknown      `json:"unknowns"`
	Verifications []Verification `json:"verifications"`
}

type Proof struct {
	Algorithm string `json:"algorithm"`
	Digest    string `json:"digest"`
}

type Certificate struct {
	Version        string         `json:"version"`
	DecisionID     string         `json:"decision_id"`
	RunID          string         `json:"run_id"`
	Recommendation string         `json:"recommendation"`
	Verdict        Verdict        `json:"verdict"`
	ActionAllowed  bool           `json:"action_allowed"`
	HumanApproved  bool           `json:"human_approved"`
	Conditions     []string       `json:"conditions"`
	PolicyVersion  string         `json:"policy_version"`
	ArtifactHashes []string       `json:"artifact_hashes"`
	Claims         []Claim        `json:"claims"`
	Verifications  []Verification `json:"verifications"`
	IssuedAt       time.Time      `json:"issued_at"`
	Proof          Proof          `json:"proof"`
}

type Analysis struct {
	Claims      []Claim      `json:"claims"`
	Evidence    []Evidence   `json:"evidence"`
	Relations   []Relation   `json:"relations"`
	Assumptions []Assumption `json:"assumptions"`
	Unknowns    []Unknown    `json:"unknowns"`
}

type PolicyDefinition struct {
	ID         string          `json:"id"`
	Version    string          `json:"version"`
	Definition json.RawMessage `json:"definition"`
	Active     bool            `json:"active"`
	CreatedAt  time.Time       `json:"created_at"`
}

type LifecycleEvent struct {
	RunID      string          `json:"run_id"`
	Type       string          `json:"type"`
	Data       json.RawMessage `json:"data"`
	OccurredAt time.Time       `json:"occurred_at"`
}
