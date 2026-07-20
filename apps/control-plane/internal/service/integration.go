package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

type CreateConnectionInput struct {
	Provider   string `json:"provider"`
	Repository string `json:"repository"`
	Endpoint   string `json:"endpoint"`
}

type CreateConnectionResult struct {
	Connection domain.ProjectConnection `json:"connection"`
	Token      string                   `json:"token"`
	Workflow   string                   `json:"workflow"`
}

func (s *Service) CreateProjectConnection(ctx context.Context, projectID string, input CreateConnectionInput) (CreateConnectionResult, error) {
	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return CreateConnectionResult{}, err
	}
	if input.Provider == "" {
		input.Provider = "github-actions"
	}
	if input.Provider != "github-actions" {
		return CreateConnectionResult{}, fmt.Errorf("%w: only github-actions connections are supported", ErrInvalid)
	}
	if input.Repository == "" {
		input.Repository = project.Repository
	}
	if strings.TrimSpace(input.Repository) == "" {
		return CreateConnectionResult{}, fmt.Errorf("%w: repository is required", ErrInvalid)
	}
	if input.Endpoint == "" {
		input.Endpoint = "https://epistemic.example.com"
	}
	token, err := newIngestToken()
	if err != nil {
		return CreateConnectionResult{}, err
	}
	now := s.now()
	connection := domain.ProjectConnection{
		ID: newID("con"), AccountID: project.AccountID, ProjectID: project.ID, Provider: input.Provider,
		Repository: strings.TrimSpace(input.Repository), Status: "active", TokenHash: tokenDigest(token),
		TokenPrefix: token[:12], CreatedAt: now,
	}
	if err = s.repo.CreateProjectConnection(ctx, connection); err != nil {
		return CreateConnectionResult{}, err
	}
	workflow := fmt.Sprintf(`- name: Epistemic quality gate
  uses: OlegGitH/epistemic-engine/adapters/github-action@v0.2
  with:
    config: .epistemic.yaml
    certificate: .epistemic/certificate.json
    report: .epistemic/project-quality.json
    endpoint: %s
    token: ${{ secrets.EPISTEMIC_INGEST_TOKEN }}`, strings.TrimRight(input.Endpoint, "/"))
	return CreateConnectionResult{Connection: connection, Token: token, Workflow: workflow}, nil
}

func (s *Service) RevokeProjectConnection(ctx context.Context, connectionID string) (domain.ProjectConnection, error) {
	return s.repo.RevokeProjectConnection(ctx, connectionID, s.now())
}

type IngestContext struct {
	Repository string `json:"repository"`
	CommitSHA  string `json:"commit_sha"`
	Branch     string `json:"branch"`
	Workflow   string `json:"workflow"`
	RunURL     string `json:"run_url"`
}

type IngestInput struct {
	ExternalID    string          `json:"external_id"`
	AISystemID    string          `json:"ai_system_id"`
	PolicyVersion string          `json:"policy_version"`
	Context       IngestContext   `json:"context"`
	Report        json.RawMessage `json:"report"`
	Certificate   json.RawMessage `json:"certificate"`
}

type IngestResult struct {
	ConnectionID  string `json:"connection_id"`
	ProjectID     string `json:"project_id"`
	ReportID      string `json:"report_id,omitempty"`
	CertificateID string `json:"certificate_id,omitempty"`
	Status        string `json:"status"`
}

func (s *Service) IngestProject(ctx context.Context, token string, input IngestInput) (IngestResult, error) {
	if token == "" {
		return IngestResult{}, ErrUnauthorized
	}
	connection, err := s.repo.GetProjectConnectionByTokenHash(ctx, tokenDigest(token))
	if err != nil {
		return IngestResult{}, ErrUnauthorized
	}
	if strings.TrimSpace(input.ExternalID) == "" {
		return IngestResult{}, fmt.Errorf("%w: external_id is required for idempotency", ErrInvalid)
	}
	if len(input.Report) == 0 && len(input.Certificate) == 0 {
		return IngestResult{}, fmt.Errorf("%w: report or certificate is required", ErrInvalid)
	}
	if input.AISystemID != "" {
		system, systemErr := s.repo.GetAISystem(ctx, input.AISystemID)
		if systemErr != nil {
			return IngestResult{}, systemErr
		}
		if system.ProjectID != connection.ProjectID {
			return IngestResult{}, fmt.Errorf("%w: AI system does not belong to connected project", ErrInvalid)
		}
	}
	now := s.now()
	ingest := domain.ProjectIngest{Connection: connection}
	result := IngestResult{ConnectionID: connection.ID, ProjectID: connection.ProjectID, Status: "accepted"}
	if len(input.Report) > 0 && string(input.Report) != "null" {
		var value struct {
			Tool     string `json:"tool"`
			Status   string `json:"status"`
			ExitCode int    `json:"exit_code"`
			Summary  string `json:"summary"`
		}
		if err = json.Unmarshal(input.Report, &value); err != nil {
			return IngestResult{}, fmt.Errorf("%w: invalid report: %v", ErrInvalid, err)
		}
		if value.Tool == "" {
			value.Tool = "project-quality"
		}
		if !validReportStatus(value.Status) {
			return IngestResult{}, fmt.Errorf("%w: unsupported report status", ErrInvalid)
		}
		report := domain.ProjectReport{
			ID: newID("rpt"), ExternalID: input.ExternalID, AccountID: connection.AccountID, ProjectID: connection.ProjectID,
			ConnectionID: connection.ID, AISystemID: input.AISystemID, Tool: value.Tool, Status: value.Status,
			ExitCode: value.ExitCode, Summary: value.Summary, Repository: input.Context.Repository, CommitSHA: input.Context.CommitSHA,
			Branch: input.Context.Branch, Workflow: input.Context.Workflow, RunURL: input.Context.RunURL, Details: input.Report, ReceivedAt: now,
		}
		ingest.Report = &report
		result.ReportID = report.ID
	}
	if len(input.Certificate) > 0 && string(input.Certificate) != "null" {
		certificate, certificateErr := validatePortableCertificate(input.Certificate)
		if certificateErr != nil {
			return IngestResult{}, fmt.Errorf("%w: invalid certificate: %v", ErrInvalid, certificateErr)
		}
		var decisionResult epistemic.DecisionResult
		if err = json.Unmarshal(certificate.Result, &decisionResult); err != nil {
			return IngestResult{}, fmt.Errorf("%w: invalid certificate result: %v", ErrInvalid, err)
		}
		policyVersion := input.PolicyVersion
		if policyVersion == "" {
			policyVersion = "epistemic.dev/v0.1"
		}
		publication := domain.PublishedCertificate{
			ID: newID("pub"), ExternalID: input.ExternalID, AccountID: connection.AccountID, ProjectID: connection.ProjectID,
			ConnectionID: connection.ID, AISystemID: input.AISystemID, DecisionID: certificate.DecisionID,
			RunID: decisionResult.Context.RunID, Status: decisionResult.Status, ActionAllowed: decisionResult.ActionAllowed,
			PolicyVersion: policyVersion, Digest: certificate.Proof.Digest, ArtifactHashes: certificate.ArtifactHashes,
			IssuedAt: certificate.IssuedAt, ReceivedAt: now, Raw: input.Certificate,
		}
		ingest.Certificate = &publication
		result.CertificateID = publication.ID
	}
	stored, err := s.repo.SaveProjectIngest(ctx, ingest)
	if err != nil {
		return IngestResult{}, err
	}
	if stored.Report != nil {
		result.ReportID = stored.Report.ID
	}
	if stored.Certificate != nil {
		result.CertificateID = stored.Certificate.ID
	}
	return result, nil
}

func validatePortableCertificate(raw json.RawMessage) (epistemic.Certificate, error) {
	var certificate epistemic.Certificate
	if err := json.Unmarshal(raw, &certificate); err != nil {
		return certificate, err
	}
	if certificate.DecisionID == "" || certificate.Proof.Algorithm != "SHA-256" || certificate.Proof.Digest == "" || certificate.IssuedAt.IsZero() {
		return certificate, fmt.Errorf("required certificate identity, timestamp, or proof is missing")
	}
	expected := certificate.Proof.Digest
	certificate.Proof.Digest = ""
	actual, err := epistemic.Hash(certificate)
	certificate.Proof.Digest = expected
	if err != nil {
		return certificate, err
	}
	if actual != expected {
		return certificate, fmt.Errorf("proof digest mismatch")
	}
	return certificate, nil
}

func newIngestToken() (string, error) {
	var value [32]byte
	if _, err := rand.Read(value[:]); err != nil {
		return "", err
	}
	return "epk_" + base64.RawURLEncoding.EncodeToString(value[:]), nil
}

func tokenDigest(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func validReportStatus(status string) bool {
	switch status {
	case "passed", "failed", "error", "pending", "indeterminate":
		return true
	default:
		return false
	}
}
