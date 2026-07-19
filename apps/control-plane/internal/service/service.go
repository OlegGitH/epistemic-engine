package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/analysis"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/lifecycle"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/policy"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/store"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/verification"
)

var (
	ErrInvalid      = errors.New("invalid request")
	ErrConflict     = errors.New("conflict")
	ErrUnauthorized = errors.New("unauthorized")
)

type Service struct {
	repo     store.Repository
	analyzer analysis.Analyzer
	executor verification.Runner
	bus      *lifecycle.Bus
	now      func() time.Time
}

type Option func(*Service)

func WithExecutor(executor verification.Runner) Option {
	return func(s *Service) { s.executor = executor }
}
func WithLifecycleBus(bus *lifecycle.Bus) Option { return func(s *Service) { s.bus = bus } }

func New(repo store.Repository, analyzer analysis.Analyzer, options ...Option) *Service {
	s := &Service{repo: repo, analyzer: analyzer, executor: verification.DisabledRunner{}, bus: lifecycle.NewBus(), now: func() time.Time { return time.Now().UTC() }}
	for _, option := range options {
		option(s)
	}
	return s
}

type CreateRunInput struct {
	AccountID       string          `json:"account_id"`
	ProjectID       string          `json:"project_id"`
	AISystemID      string          `json:"ai_system_id"`
	ExternalTraceID string          `json:"external_trace_id"`
	Title           string          `json:"title"`
	Goal            string          `json:"goal"`
	Source          string          `json:"source"`
	Recommendation  string          `json:"recommendation"`
	ActionType      string          `json:"action_type"`
	Subject         string          `json:"subject"`
	RiskLevel       string          `json:"risk_level"`
	Raw             json.RawMessage `json:"raw,omitempty"`
}

func (s *Service) CreateRun(ctx context.Context, in CreateRunInput) (domain.Run, error) {
	if strings.TrimSpace(in.Recommendation) == "" {
		return domain.Run{}, fmt.Errorf("%w: recommendation is required", ErrInvalid)
	}
	now := s.now()
	runID := newID("run")
	decisionID := newID("dec")
	if in.Title == "" {
		in.Title = "Deployment readiness review"
	}
	if in.Source == "" {
		in.Source = "api"
	}
	if in.Goal == "" {
		in.Goal = "Determine whether the proposed change is sufficiently verified to deploy."
	}
	if in.ActionType == "" {
		in.ActionType = "software_deployment"
	}
	if in.Subject == "" {
		in.Subject = in.Title
	}
	if in.RiskLevel == "" {
		in.RiskLevel = "high"
	}
	if in.ProjectID != "" {
		project, err := s.repo.GetProject(ctx, in.ProjectID)
		if err != nil {
			return domain.Run{}, err
		}
		if in.AccountID != "" && in.AccountID != project.AccountID {
			return domain.Run{}, fmt.Errorf("%w: project does not belong to account", ErrInvalid)
		}
		in.AccountID = project.AccountID
	}
	if in.AccountID != "" {
		if _, err := s.repo.GetAccount(ctx, in.AccountID); err != nil {
			return domain.Run{}, err
		}
	}
	if in.AISystemID != "" {
		system, err := s.repo.GetAISystem(ctx, in.AISystemID)
		if err != nil {
			return domain.Run{}, err
		}
		if in.ProjectID == "" || system.ProjectID != in.ProjectID {
			return domain.Run{}, fmt.Errorf("%w: AI system does not belong to project", ErrInvalid)
		}
	}
	run := domain.Run{ID: runID, AccountID: in.AccountID, ProjectID: in.ProjectID, AISystemID: in.AISystemID, ExternalTraceID: in.ExternalTraceID, Title: in.Title, Goal: in.Goal, Source: in.Source, Recommendation: in.Recommendation, Status: domain.RunIngesting, DecisionID: decisionID, Events: []domain.Event{}, Raw: in.Raw, CreatedAt: now}
	decision := domain.Decision{ID: decisionID, RunID: runID, Recommendation: in.Recommendation, ActionType: in.ActionType, Subject: in.Subject, RiskLevel: in.RiskLevel, PolicyVersion: policy.Version, Conditions: []string{}, ClaimIDs: []string{}, CreatedAt: now}
	if err := s.repo.CreateRun(ctx, run, decision); err != nil {
		return domain.Run{}, err
	}
	s.publish(run.ID, "run.created", run)
	return run, nil
}

type AddEventInput struct {
	ExternalID    string          `json:"external_id"`
	Sequence      int64           `json:"sequence"`
	Type          string          `json:"type"`
	Source        string          `json:"source"`
	CorrelationID string          `json:"correlation_id"`
	OccurredAt    time.Time       `json:"occurred_at"`
	Payload       json.RawMessage `json:"payload"`
}

func (s *Service) AddEvent(ctx context.Context, runID string, in AddEventInput) (domain.Event, error) {
	if in.Type == "" || len(in.Payload) == 0 {
		return domain.Event{}, fmt.Errorf("%w: type and payload are required", ErrInvalid)
	}
	if in.Source == "" {
		in.Source = "agent"
	}
	if in.OccurredAt.IsZero() {
		in.OccurredAt = s.now()
	}
	e := domain.Event{ID: newID("evt"), ExternalID: in.ExternalID, Sequence: in.Sequence, Type: in.Type, Source: in.Source, CorrelationID: in.CorrelationID, OccurredAt: in.OccurredAt, Payload: in.Payload}
	stored, created, err := s.repo.AddEvent(ctx, runID, e)
	if err != nil {
		return domain.Event{}, err
	}
	if created {
		s.publish(runID, "run.event.appended", stored)
	}
	return stored, nil
}

func (s *Service) Analyze(ctx context.Context, runID string) (domain.Graph, error) {
	run, err := s.repo.GetRun(ctx, runID)
	if err != nil {
		return domain.Graph{}, err
	}
	if run.Status == domain.RunAnalyzed {
		return s.repo.GetGraphByRun(ctx, runID)
	}
	graph, err := s.repo.GetGraphByRun(ctx, runID)
	if err != nil {
		return domain.Graph{}, err
	}
	a, err := s.analyzer.Analyze(ctx, run, graph.Decision)
	if err != nil {
		return domain.Graph{}, err
	}
	now := s.now()
	idMap := map[string]string{}
	for i := range a.Evidence {
		original := a.Evidence[i].ID
		a.Evidence[i].ID = newID("evd")
		if original != "" {
			idMap[original] = a.Evidence[i].ID
		}
		if a.Evidence[i].ObservedAt.IsZero() {
			a.Evidence[i].ObservedAt = now
		}
	}
	for i := range a.Claims {
		original := a.Claims[i].ID
		a.Claims[i].ID = newID("clm")
		if original != "" {
			idMap[original] = a.Claims[i].ID
		}
		a.Claims[i].DecisionID = graph.Decision.ID
		a.Claims[i].CreatedAt = now
		if a.Claims[i].Importance == "" {
			if a.Claims[i].Critical {
				a.Claims[i].Importance = "critical"
			} else {
				a.Claims[i].Importance = "supporting"
			}
		}
		if a.Claims[i].Support.Semantics == "" {
			a.Claims[i].Support.Semantics = "explainable evidence support; not probability of truth"
		}
		if len(a.Relations) == 0 && (a.Claims[i].State == domain.ClaimSupported || a.Claims[i].State == domain.ClaimContradicted) && len(a.Evidence) > 0 {
			for _, e := range a.Evidence {
				a.Claims[i].EvidenceIDs = append(a.Claims[i].EvidenceIDs, e.ID)
				relType := domain.RelationSupports
				if a.Claims[i].State == domain.ClaimContradicted {
					relType = domain.RelationContradicts
				}
				a.Relations = append(a.Relations, domain.Relation{ID: newID("rel"), FromID: e.ID, ToID: a.Claims[i].ID, Type: relType, Rationale: a.Claims[i].Justification})
			}
		}
	}
	for i := range a.Relations {
		if mapped := idMap[a.Relations[i].FromID]; mapped != "" {
			a.Relations[i].FromID = mapped
		}
		if mapped := idMap[a.Relations[i].ToID]; mapped != "" {
			a.Relations[i].ToID = mapped
		}
		if a.Relations[i].ID == "" {
			a.Relations[i].ID = newID("rel")
		}
	}
	for i := range a.Claims {
		for j, evidenceID := range a.Claims[i].EvidenceIDs {
			if mapped := idMap[evidenceID]; mapped != "" {
				a.Claims[i].EvidenceIDs[j] = mapped
			}
		}
	}
	for i := range a.Assumptions {
		a.Assumptions[i].ID = newID("asm")
		a.Assumptions[i].DecisionID = graph.Decision.ID
		if mapped := idMap[a.Assumptions[i].ClaimID]; mapped != "" {
			a.Assumptions[i].ClaimID = mapped
		}
	}
	for i := range a.Unknowns {
		a.Unknowns[i].ID = newID("unk")
		a.Unknowns[i].DecisionID = graph.Decision.ID
	}
	if err := s.repo.SaveAnalysis(ctx, runID, a); err != nil {
		return domain.Graph{}, err
	}
	result, err := s.repo.GetGraphByRun(ctx, runID)
	if err == nil {
		s.publish(runID, "analysis.completed", map[string]any{"claims": len(result.Claims), "evidence": len(result.Evidence), "relations": len(result.Relations)})
	}
	return result, err
}

func (s *Service) Graph(ctx context.Context, runID string) (domain.Graph, error) {
	return s.repo.GetGraphByRun(ctx, runID)
}

func (s *Service) Plan(ctx context.Context, decisionID string) ([]domain.Verification, error) {
	g, err := s.repo.GetGraphByDecision(ctx, decisionID)
	if err != nil {
		return nil, err
	}
	if len(g.Verifications) > 0 {
		return g.Verifications, nil
	}
	var out []domain.Verification
	for _, c := range g.Claims {
		if !c.Critical || c.State == domain.ClaimSupported || c.State == domain.ClaimExternallyVerified {
			continue
		}
		kind, specification := verification.SpecificationForClaim(c)
		out = append(out, domain.Verification{ID: newID("ver"), DecisionID: decisionID, ClaimID: c.ID, Check: "Create and run the smallest targeted check for: " + c.Statement, Kind: kind, Specification: specification, Environment: "sandbox", Status: "planned", RequiresApproval: true, CreatedAt: s.now()})
	}
	if len(out) == 0 {
		return []domain.Verification{}, nil
	}
	if err := s.repo.SaveVerifications(ctx, out); err != nil {
		return nil, err
	}
	s.publish(g.Run.ID, "verification.plan.created", out)
	return out, nil
}

type ExecuteVerificationInput struct {
	Environment string          `json:"environment"`
	Outcome     string          `json:"outcome"`
	Artifact    json.RawMessage `json:"artifact"`
	Approved    bool            `json:"approved"`
	ApprovedBy  string          `json:"approved_by"`
}

func (s *Service) ExecuteVerification(ctx context.Context, id string, in ExecuteVerificationInput) (domain.Verification, error) {
	v, err := s.repo.GetVerification(ctx, id)
	if err != nil {
		return v, err
	}
	if v.Status == "completed" {
		return v, fmt.Errorf("%w: verification result is immutable", ErrConflict)
	}
	if in.Environment == "" {
		in.Environment = v.Environment
	}
	if in.Environment != "sandbox" && in.Environment != "staging" {
		return v, fmt.Errorf("%w: verification environment must be sandbox or staging", ErrInvalid)
	}
	if v.RequiresApproval && !in.Approved {
		return v, fmt.Errorf("%w: explicit human approval is required before verification execution", ErrInvalid)
	}
	if in.ApprovedBy == "" {
		in.ApprovedBy = "human-reviewer"
	}
	v.Approved, v.ApprovedBy = true, in.ApprovedBy
	if in.Outcome == "" && len(in.Artifact) == 0 {
		v.Status = "running"
		_ = s.repo.UpdateVerification(ctx, v)
		result, executeErr := s.executor.Execute(ctx, v)
		if executeErr != nil {
			return v, executeErr
		}
		in.Outcome, in.Artifact = result.Outcome, result.Artifact
	}
	if in.Outcome != "passed" && in.Outcome != "failed" {
		return v, fmt.Errorf("%w: outcome must be passed or failed", ErrInvalid)
	}
	if len(in.Artifact) == 0 {
		return v, fmt.Errorf("%w: a verification artifact is required", ErrInvalid)
	}
	now := s.now()
	v.Environment = in.Environment
	v.Outcome = in.Outcome
	v.Artifact = in.Artifact
	sum := sha256.Sum256(in.Artifact)
	v.ArtifactHash = hex.EncodeToString(sum[:])
	v.Status = "completed"
	v.ExecutedAt = &now
	if err := s.repo.UpdateVerification(ctx, v); err != nil {
		return v, err
	}
	g, graphErr := s.repo.GetGraphByDecision(ctx, v.DecisionID)
	if graphErr == nil {
		s.publish(g.Run.ID, "verification.completed", v)
	}
	return v, nil
}

func (s *Service) Evaluate(ctx context.Context, decisionID string, humanApproved bool) (domain.Certificate, error) {
	if cert, err := s.repo.GetCertificate(ctx, decisionID); err == nil {
		return cert, nil
	}
	g, err := s.repo.GetGraphByDecision(ctx, decisionID)
	if err != nil {
		return domain.Certificate{}, err
	}
	result := policy.Evaluate(g, humanApproved)
	if result.Conditions == nil {
		result.Conditions = []string{}
	}
	now := s.now()
	d := g.Decision
	d.Verdict = result.Verdict
	d.Conditions = result.Conditions
	d.HumanApproved = humanApproved
	d.ActionAllowed = result.ActionAllowed
	d.EvaluatedAt = &now
	if err := s.repo.SaveDecision(ctx, d); err != nil {
		return domain.Certificate{}, err
	}
	artifactHashes := make([]string, 0, len(g.Verifications)+len(g.Evidence))
	for _, evidence := range g.Evidence {
		if evidence.ContentHash != "" {
			artifactHashes = append(artifactHashes, evidence.ContentHash)
		}
	}
	for _, verification := range g.Verifications {
		if verification.ArtifactHash != "" {
			artifactHashes = append(artifactHashes, verification.ArtifactHash)
		}
	}
	sort.Strings(artifactHashes)
	cert := domain.Certificate{Version: "1.0", DecisionID: d.ID, RunID: d.RunID, Recommendation: d.Recommendation, Verdict: d.Verdict, ActionAllowed: d.ActionAllowed, HumanApproved: d.HumanApproved, Conditions: d.Conditions, PolicyVersion: d.PolicyVersion, ArtifactHashes: artifactHashes, Claims: g.Claims, Verifications: g.Verifications, IssuedAt: now}
	digest, err := CertificateDigest(cert)
	if err != nil {
		return domain.Certificate{}, err
	}
	cert.Proof = domain.Proof{Algorithm: "SHA-256", Digest: digest}
	if err := s.repo.SaveCertificate(ctx, cert); err != nil {
		return domain.Certificate{}, err
	}
	s.publish(d.RunID, "decision.evaluated", cert)
	return cert, nil
}

func (s *Service) Certificate(ctx context.Context, decisionID string) (domain.Certificate, error) {
	return s.repo.GetCertificate(ctx, decisionID)
}

func (s *Service) Subscribe(runID string) (<-chan domain.LifecycleEvent, func()) {
	return s.bus.Subscribe(runID)
}

func (s *Service) publish(runID, eventType string, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		return
	}
	s.bus.Publish(domain.LifecycleEvent{RunID: runID, Type: eventType, Data: data, OccurredAt: s.now()})
}

// CertificateDigest reproduces the proof by hashing the stable certificate
// payload with the proof field omitted.
func CertificateDigest(cert domain.Certificate) (string, error) {
	cert.Proof = domain.Proof{}
	unsigned, err := json.Marshal(cert)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(unsigned)
	return hex.EncodeToString(sum[:]), nil
}

func newID(prefix string) string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}
