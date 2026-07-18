package store

import (
	"context"
	"sort"
	"sync"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

type Memory struct {
	mu            sync.RWMutex
	runs          map[string]domain.Run
	decisions     map[string]domain.Decision
	claims        map[string]domain.Claim
	evidence      map[string]domain.Evidence
	relations     map[string]domain.Relation
	assumptions   map[string]domain.Assumption
	unknowns      map[string]domain.Unknown
	verifications map[string]domain.Verification
	certificates  map[string]domain.Certificate
}

func NewMemory() *Memory {
	return &Memory{
		runs: map[string]domain.Run{}, decisions: map[string]domain.Decision{}, claims: map[string]domain.Claim{},
		evidence: map[string]domain.Evidence{}, relations: map[string]domain.Relation{}, assumptions: map[string]domain.Assumption{},
		unknowns: map[string]domain.Unknown{}, verifications: map[string]domain.Verification{}, certificates: map[string]domain.Certificate{},
	}
}

func (m *Memory) CreateRun(_ context.Context, run domain.Run, decision domain.Decision) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs[run.ID] = run
	m.decisions[decision.ID] = decision
	return nil
}
func (m *Memory) GetRun(_ context.Context, id string) (domain.Run, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.runs[id]
	if !ok {
		return v, ErrNotFound
	}
	return v, nil
}
func (m *Memory) AddEvent(_ context.Context, id string, event domain.Event) (domain.Event, bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.runs[id]
	if !ok {
		return event, false, ErrNotFound
	}
	for _, existing := range v.Events {
		if event.ExternalID != "" && existing.ExternalID == event.ExternalID {
			return existing, false, nil
		}
	}
	if event.Sequence == 0 {
		event.Sequence = int64(len(v.Events) + 1)
	}
	v.Events = append(v.Events, event)
	m.runs[id] = v
	return event, true, nil
}
func (m *Memory) SaveAnalysis(_ context.Context, runID string, a domain.Analysis) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.runs[runID]
	if !ok {
		return ErrNotFound
	}
	r.Status = domain.RunAnalyzed
	m.runs[runID] = r
	for _, v := range a.Claims {
		m.claims[v.ID] = v
	}
	for _, v := range a.Evidence {
		m.evidence[v.ID] = v
	}
	for _, v := range a.Relations {
		m.relations[v.ID] = v
	}
	for _, v := range a.Assumptions {
		m.assumptions[v.ID] = v
	}
	for _, v := range a.Unknowns {
		m.unknowns[v.ID] = v
	}
	d := m.decisions[r.DecisionID]
	d.ClaimIDs = nil
	for _, v := range a.Claims {
		d.ClaimIDs = append(d.ClaimIDs, v.ID)
	}
	m.decisions[d.ID] = d
	return nil
}
func (m *Memory) GetGraphByRun(_ context.Context, id string) (domain.Graph, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.runs[id]
	if !ok {
		return domain.Graph{}, ErrNotFound
	}
	return m.graph(r), nil
}
func (m *Memory) GetGraphByDecision(_ context.Context, id string) (domain.Graph, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	d, ok := m.decisions[id]
	if !ok {
		return domain.Graph{}, ErrNotFound
	}
	r := m.runs[d.RunID]
	return m.graph(r), nil
}
func (m *Memory) graph(r domain.Run) domain.Graph {
	d := m.decisions[r.DecisionID]
	g := domain.Graph{Run: r, Decision: d}
	objectIDs := map[string]bool{d.ID: true, r.ID: true}
	for _, v := range m.claims {
		if v.DecisionID == d.ID {
			g.Claims = append(g.Claims, v)
			objectIDs[v.ID] = true
		}
	}
	for _, v := range m.evidence {
		if v.RunID == r.ID {
			g.Evidence = append(g.Evidence, v)
			objectIDs[v.ID] = true
		}
	}
	for _, v := range m.assumptions {
		if v.DecisionID == d.ID {
			g.Assumptions = append(g.Assumptions, v)
		}
	}
	for _, v := range m.unknowns {
		if v.DecisionID == d.ID {
			g.Unknowns = append(g.Unknowns, v)
		}
	}
	for _, v := range m.verifications {
		if v.DecisionID == d.ID {
			g.Verifications = append(g.Verifications, v)
			objectIDs[v.ID] = true
		}
	}
	for _, v := range m.relations {
		if objectIDs[v.FromID] && objectIDs[v.ToID] {
			g.Relations = append(g.Relations, v)
		}
	}
	sort.Slice(g.Claims, func(i, j int) bool { return g.Claims[i].ID < g.Claims[j].ID })
	sort.Slice(g.Evidence, func(i, j int) bool { return g.Evidence[i].ID < g.Evidence[j].ID })
	sort.Slice(g.Relations, func(i, j int) bool { return g.Relations[i].ID < g.Relations[j].ID })
	sort.Slice(g.Assumptions, func(i, j int) bool { return g.Assumptions[i].ID < g.Assumptions[j].ID })
	sort.Slice(g.Unknowns, func(i, j int) bool { return g.Unknowns[i].ID < g.Unknowns[j].ID })
	sort.Slice(g.Verifications, func(i, j int) bool { return g.Verifications[i].ID < g.Verifications[j].ID })
	return g
}
func (m *Memory) SaveVerifications(_ context.Context, values []domain.Verification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, v := range values {
		m.verifications[v.ID] = v
	}
	return nil
}
func (m *Memory) GetVerification(_ context.Context, id string) (domain.Verification, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.verifications[id]
	if !ok {
		return v, ErrNotFound
	}
	return v, nil
}
func (m *Memory) UpdateVerification(_ context.Context, v domain.Verification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.verifications[v.ID]; !ok {
		return ErrNotFound
	}
	m.verifications[v.ID] = v
	if v.Outcome == "passed" {
		c := m.claims[v.ClaimID]
		c.State = domain.ClaimExternallyVerified
		c.Support.DirectVerificationStrength = 1
		c.Support.Value = 1
		m.claims[c.ID] = c
		m.relations["rel_verified_"+v.ID] = domain.Relation{ID: "rel_verified_" + v.ID, FromID: c.ID, ToID: v.ID, Type: domain.RelationVerifiedBy, Rationale: "Controlled verification passed; artifact " + v.ArtifactHash}
		for id, u := range m.unknowns {
			if u.DecisionID == v.DecisionID && u.Critical {
				u.Resolved = true
				m.unknowns[id] = u
			}
		}
	}
	if v.Outcome == "failed" {
		c := m.claims[v.ClaimID]
		c.State = domain.ClaimContradicted
		c.Support.ContradictionBurden = 1
		c.Support.Value = 0
		m.claims[c.ID] = c
		m.relations["rel_verified_"+v.ID] = domain.Relation{ID: "rel_verified_" + v.ID, FromID: c.ID, ToID: v.ID, Type: domain.RelationVerifiedBy, Rationale: "Controlled verification failed; artifact " + v.ArtifactHash}
	}
	return nil
}
func (m *Memory) SaveDecision(_ context.Context, v domain.Decision) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.decisions[v.ID]; !ok {
		return ErrNotFound
	}
	m.decisions[v.ID] = v
	return nil
}
func (m *Memory) GetCertificate(_ context.Context, id string) (domain.Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	v, ok := m.certificates[id]
	if !ok {
		return v, ErrNotFound
	}
	return v, nil
}
func (m *Memory) SaveCertificate(_ context.Context, v domain.Certificate) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, exists := m.certificates[v.DecisionID]; exists {
		return nil
	}
	m.certificates[v.DecisionID] = v
	return nil
}
