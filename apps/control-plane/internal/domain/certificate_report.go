package domain

import "time"

type CertificateReportCounts struct {
	CriticalClaims      int `json:"critical_claims"`
	SupportedClaims     int `json:"supported_claims"`
	ContradictedClaims  int `json:"contradicted_claims"`
	OpenClaims          int `json:"open_claims"`
	VerificationChecks  int `json:"verification_checks"`
	PassedVerifications int `json:"passed_verifications"`
	EvidenceArtifacts   int `json:"evidence_artifacts"`
}

type CertificateReportClaim struct {
	Statement      string   `json:"statement"`
	State          string   `json:"state"`
	Assessment     string   `json:"assessment"`
	SupportPercent int      `json:"support_percent"`
	EvidenceCount  int      `json:"evidence_count"`
	Required       []string `json:"required_evidence"`
}

type CertificateReportVerification struct {
	Check        string `json:"check"`
	Kind         string `json:"kind"`
	Outcome      string `json:"outcome"`
	Environment  string `json:"environment"`
	ApprovedBy   string `json:"approved_by,omitempty"`
	ArtifactHash string `json:"artifact_hash,omitempty"`
}

type HumanCertificateReport struct {
	Version               string                          `json:"version"`
	DecisionID            string                          `json:"decision_id"`
	RunID                 string                          `json:"run_id"`
	Decision              string                          `json:"decision"`
	Headline              string                          `json:"headline"`
	Summary               string                          `json:"summary"`
	Recommendation        string                          `json:"recommendation"`
	Verdict               string                          `json:"verdict"`
	ActionAllowed         bool                            `json:"action_allowed"`
	HumanApprovalRequired bool                            `json:"human_approval_required"`
	HumanApprovalGranted  bool                            `json:"human_approval_granted"`
	PolicyVersion         string                          `json:"policy_version"`
	IssuedAt              time.Time                       `json:"issued_at"`
	Counts                CertificateReportCounts         `json:"counts"`
	CriticalClaims        []CertificateReportClaim        `json:"critical_claims"`
	Conditions            []string                        `json:"conditions"`
	Verifications         []CertificateReportVerification `json:"verifications"`
	Proof                 Proof                           `json:"proof"`
	Markdown              string                          `json:"markdown"`
}
