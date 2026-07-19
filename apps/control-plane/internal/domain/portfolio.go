package domain

import "time"

type Account struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	CreatedAt time.Time `json:"created_at"`
}

type Project struct {
	ID         string    `json:"id"`
	AccountID  string    `json:"account_id"`
	Name       string    `json:"name"`
	Slug       string    `json:"slug"`
	Repository string    `json:"repository,omitempty"`
	Owner      string    `json:"owner,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type AISystem struct {
	ID          string     `json:"id"`
	AccountID   string     `json:"account_id"`
	ProjectID   string     `json:"project_id"`
	Name        string     `json:"name"`
	Provider    string     `json:"provider"`
	Model       string     `json:"model"`
	Purpose     string     `json:"purpose"`
	DataClasses []string   `json:"data_classes"`
	Tools       []string   `json:"tools"`
	Owner       string     `json:"owner,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
}

type DashboardMetrics struct {
	Projects             int `json:"projects"`
	ConnectedProjects    int `json:"connected_projects"`
	AISystems            int `json:"ai_systems"`
	Reports              int `json:"reports"`
	ValidCertificates    int `json:"valid_certificates"`
	AttentionItems       int `json:"attention_items"`
	KnowledgeCoveragePct int `json:"knowledge_coverage_pct"`
}

type KnowledgeSummary struct {
	Claims             int `json:"claims"`
	SupportedClaims    int `json:"supported_claims"`
	ContradictedClaims int `json:"contradicted_claims"`
	EvidenceArtifacts  int `json:"evidence_artifacts"`
	OpenUnknowns       int `json:"open_unknowns"`
	StaleClaims        int `json:"stale_claims"`
}

type ProjectSummary struct {
	Project
	Runs                 int        `json:"runs"`
	AISystems            int        `json:"ai_systems"`
	Claims               int        `json:"claims"`
	SupportedClaims      int        `json:"supported_claims"`
	EvidenceArtifacts    int        `json:"evidence_artifacts"`
	OpenUnknowns         int        `json:"open_unknowns"`
	KnowledgeCoveragePct int        `json:"knowledge_coverage_pct"`
	CertificationStatus  string     `json:"certification_status"`
	ConnectionStatus     string     `json:"connection_status"`
	Reports              int        `json:"reports"`
	LatestRunID          string     `json:"latest_run_id,omitempty"`
	LastActivityAt       *time.Time `json:"last_activity_at,omitempty"`
}

type AISystemSummary struct {
	AISystem
	ProjectName         string     `json:"project_name"`
	CertificationStatus string     `json:"certification_status"`
	CertificateDigest   string     `json:"certificate_digest,omitempty"`
	LastEvaluatedAt     *time.Time `json:"last_evaluated_at,omitempty"`
}

type CertificateSummary struct {
	DecisionID    string    `json:"decision_id"`
	RunID         string    `json:"run_id"`
	ProjectID     string    `json:"project_id"`
	ProjectName   string    `json:"project_name"`
	AISystemID    string    `json:"ai_system_id,omitempty"`
	AISystemName  string    `json:"ai_system_name,omitempty"`
	Verdict       string    `json:"verdict"`
	ActionAllowed bool      `json:"action_allowed"`
	PolicyVersion string    `json:"policy_version"`
	IssuedAt      time.Time `json:"issued_at"`
	Digest        string    `json:"digest"`
	Source        string    `json:"source"`
	ReceivedAt    time.Time `json:"received_at"`
}

type DashboardActivity struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	Title      string    `json:"title"`
	Detail     string    `json:"detail"`
	Status     string    `json:"status"`
	OccurredAt time.Time `json:"occurred_at"`
	RunID      string    `json:"run_id,omitempty"`
}

type AccountDashboard struct {
	Account      Account              `json:"account"`
	Metrics      DashboardMetrics     `json:"metrics"`
	Knowledge    KnowledgeSummary     `json:"knowledge"`
	Projects     []ProjectSummary     `json:"projects"`
	AISystems    []AISystemSummary    `json:"ai_systems"`
	Connections  []ConnectionSummary  `json:"connections"`
	Reports      []ProjectReport      `json:"reports"`
	Certificates []CertificateSummary `json:"certificates"`
	Activity     []DashboardActivity  `json:"activity"`
	GeneratedAt  time.Time            `json:"generated_at"`
}
