package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct{ pool *pgxpool.Pool }

func NewPostgres(ctx context.Context, databaseURL string) (*Postgres, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Postgres{pool: pool}, nil
}

func (p *Postgres) Close() { p.pool.Close() }

func (p *Postgres) CreateRun(ctx context.Context, run domain.Run, decision domain.Decision) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `INSERT INTO runs(id,account_id,project_id,ai_system_id,external_trace_id,title,goal,source,recommendation,status,raw,created_at) VALUES($1,NULLIF($2,''),NULLIF($3,''),NULLIF($4,''),NULLIF($5,''),$6,$7,$8,$9,$10,$11,$12)`, run.ID, run.AccountID, run.ProjectID, run.AISystemID, run.ExternalTraceID, run.Title, run.Goal, run.Source, run.Recommendation, run.Status, nullableJSON(run.Raw), run.CreatedAt)
	if err != nil {
		return err
	}
	conditions, _ := json.Marshal(decision.Conditions)
	_, err = tx.Exec(ctx, `INSERT INTO decisions(id,run_id,recommendation,action_type,subject,risk_level,policy_version,action_allowed,human_approved,conditions,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, decision.ID, decision.RunID, decision.Recommendation, decision.ActionType, decision.Subject, decision.RiskLevel, decision.PolicyVersion, decision.ActionAllowed, decision.HumanApproved, conditions, decision.CreatedAt)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE runs SET decision_id=$2 WHERE id=$1`, run.ID, decision.ID)
	if err != nil {
		return err
	}
	if run.AISystemID != "" {
		if _, err = tx.Exec(ctx, `UPDATE ai_systems SET last_used_at=$2,updated_at=$2 WHERE id=$1`, run.AISystemID, run.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (p *Postgres) GetRun(ctx context.Context, id string) (domain.Run, error) {
	var run domain.Run
	var raw []byte
	err := p.pool.QueryRow(ctx, `SELECT id,COALESCE(account_id,''),COALESCE(project_id,''),COALESCE(ai_system_id,''),COALESCE(external_trace_id,''),title,goal,source,recommendation,status,decision_id,raw,created_at FROM runs WHERE id=$1`, id).Scan(&run.ID, &run.AccountID, &run.ProjectID, &run.AISystemID, &run.ExternalTraceID, &run.Title, &run.Goal, &run.Source, &run.Recommendation, &run.Status, &run.DecisionID, &raw, &run.CreatedAt)
	if err != nil {
		return run, mapNotFound(err)
	}
	run.Raw = raw
	rows, err := p.pool.Query(ctx, `SELECT id,COALESCE(external_id,''),sequence,event_type,source,COALESCE(correlation_id,''),occurred_at,payload_json FROM run_events WHERE run_id=$1 ORDER BY sequence`, id)
	if err != nil {
		return run, err
	}
	defer rows.Close()
	for rows.Next() {
		var event domain.Event
		if err := rows.Scan(&event.ID, &event.ExternalID, &event.Sequence, &event.Type, &event.Source, &event.CorrelationID, &event.OccurredAt, &event.Payload); err != nil {
			return run, err
		}
		run.Events = append(run.Events, event)
	}
	return run, rows.Err()
}

func (p *Postgres) AddEvent(ctx context.Context, runID string, event domain.Event) (domain.Event, bool, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return event, false, err
	}
	defer tx.Rollback(ctx)
	var exists string
	if err = tx.QueryRow(ctx, `SELECT id FROM runs WHERE id=$1 FOR UPDATE`, runID).Scan(&exists); err != nil {
		return event, false, mapNotFound(err)
	}
	if event.ExternalID != "" {
		var existing domain.Event
		var payload []byte
		err = tx.QueryRow(ctx, `SELECT id,external_id,sequence,event_type,source,COALESCE(correlation_id,''),occurred_at,payload_json FROM run_events WHERE run_id=$1 AND external_id=$2`, runID, event.ExternalID).Scan(&existing.ID, &existing.ExternalID, &existing.Sequence, &existing.Type, &existing.Source, &existing.CorrelationID, &existing.OccurredAt, &payload)
		if err == nil {
			existing.Payload = payload
			return existing, false, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return event, false, err
		}
	}
	if event.Sequence == 0 {
		if err = tx.QueryRow(ctx, `SELECT COALESCE(MAX(sequence),0)+1 FROM run_events WHERE run_id=$1`, runID).Scan(&event.Sequence); err != nil {
			return event, false, err
		}
	}
	sum := sha256.Sum256(event.Payload)
	_, err = tx.Exec(ctx, `INSERT INTO run_events(id,external_id,run_id,sequence,event_type,source,correlation_id,occurred_at,payload_json,payload_sha256) VALUES($1,NULLIF($2,''),$3,$4,$5,$6,NULLIF($7,''),$8,$9,$10)`, event.ID, event.ExternalID, runID, event.Sequence, event.Type, event.Source, event.CorrelationID, event.OccurredAt, event.Payload, hex.EncodeToString(sum[:]))
	if err != nil {
		return event, false, err
	}
	return event, true, tx.Commit(ctx)
}

func (p *Postgres) SaveAnalysis(ctx context.Context, runID string, a domain.Analysis) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var decisionID string
	if err = tx.QueryRow(ctx, `SELECT decision_id FROM runs WHERE id=$1 FOR UPDATE`, runID).Scan(&decisionID); err != nil {
		return mapNotFound(err)
	}
	for _, c := range a.Claims {
		support, _ := json.Marshal(c.Support)
		required, _ := json.Marshal(c.RequiredEvidenceTypes)
		scope, _ := json.Marshal(c.Scope)
		_, err = tx.Exec(ctx, `INSERT INTO claims(id,decision_id,statement,importance,critical,scope_json,required_evidence_types,state,justification,support_score,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, c.ID, decisionID, c.Statement, c.Importance, c.Critical, scope, required, c.State, c.Justification, support, c.CreatedAt)
		if err != nil {
			return err
		}
	}
	for _, e := range a.Evidence {
		_, err = tx.Exec(ctx, `INSERT INTO evidence(id,run_id,evidence_type,source,summary,uri,content_hash,freshness_at,metadata_json,raw) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10)`, e.ID, runID, e.Kind, e.Source, e.Summary, e.URI, e.ContentHash, e.ObservedAt, nullableJSON(e.Metadata), nullableJSON(e.Raw))
		if err != nil {
			return err
		}
	}
	for _, r := range a.Relations {
		_, err = tx.Exec(ctx, `INSERT INTO relations(id,from_id,to_id,relation_type,rationale) VALUES($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`, r.ID, r.FromID, r.ToID, r.Type, r.Rationale)
		if err != nil {
			return err
		}
		if r.Type == domain.RelationSupports || r.Type == domain.RelationContradicts || r.Type == domain.RelationQualifies || r.Type == domain.RelationDerivedFrom {
			_, _ = tx.Exec(ctx, `INSERT INTO claim_evidence(claim_id,evidence_id,relation,strength,rationale) SELECT $1,$2,$3,1,$4 WHERE EXISTS(SELECT 1 FROM claims WHERE id=$1) AND EXISTS(SELECT 1 FROM evidence WHERE id=$2) ON CONFLICT DO NOTHING`, r.ToID, r.FromID, r.Type, r.Rationale)
		}
	}
	for _, a := range a.Assumptions {
		_, err = tx.Exec(ctx, `INSERT INTO assumptions(id,decision_id,claim_id,statement,critical,status,impact) VALUES($1,$2,NULLIF($3,''),$4,$5,$6,$7)`, a.ID, decisionID, a.ClaimID, a.Statement, a.Critical, a.Status, a.Impact)
		if err != nil {
			return err
		}
	}
	for _, u := range a.Unknowns {
		_, err = tx.Exec(ctx, `INSERT INTO unknowns(id,decision_id,question,critical,resolved) VALUES($1,$2,$3,$4,$5)`, u.ID, decisionID, u.Question, u.Critical, u.Resolved)
		if err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE runs SET status='analyzed' WHERE id=$1`, runID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (p *Postgres) GetGraphByRun(ctx context.Context, id string) (domain.Graph, error) {
	run, err := p.GetRun(ctx, id)
	if err != nil {
		return domain.Graph{}, err
	}
	decision, err := p.getDecision(ctx, run.DecisionID)
	if err != nil {
		return domain.Graph{}, err
	}
	return p.loadGraph(ctx, run, decision)
}
func (p *Postgres) GetGraphByDecision(ctx context.Context, id string) (domain.Graph, error) {
	decision, err := p.getDecision(ctx, id)
	if err != nil {
		return domain.Graph{}, err
	}
	run, err := p.GetRun(ctx, decision.RunID)
	if err != nil {
		return domain.Graph{}, err
	}
	return p.loadGraph(ctx, run, decision)
}

func (p *Postgres) getDecision(ctx context.Context, id string) (domain.Decision, error) {
	var d domain.Decision
	var verdict *string
	var conditions []byte
	err := p.pool.QueryRow(ctx, `SELECT id,run_id,recommendation,action_type,subject,risk_level,policy_version,verdict,action_allowed,human_approved,conditions,created_at,evaluated_at FROM decisions WHERE id=$1`, id).Scan(&d.ID, &d.RunID, &d.Recommendation, &d.ActionType, &d.Subject, &d.RiskLevel, &d.PolicyVersion, &verdict, &d.ActionAllowed, &d.HumanApproved, &conditions, &d.CreatedAt, &d.EvaluatedAt)
	if err != nil {
		return d, mapNotFound(err)
	}
	if verdict != nil {
		d.Verdict = domain.Verdict(*verdict)
	}
	_ = json.Unmarshal(conditions, &d.Conditions)
	return d, nil
}

func (p *Postgres) loadGraph(ctx context.Context, run domain.Run, decision domain.Decision) (domain.Graph, error) {
	g := domain.Graph{Run: run, Decision: decision}
	rows, err := p.pool.Query(ctx, `SELECT id,statement,importance,critical,scope_json,required_evidence_types,state,justification,support_score,created_at FROM claims WHERE decision_id=$1 ORDER BY id`, decision.ID)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var c domain.Claim
		var scope, required, support []byte
		if err = rows.Scan(&c.ID, &c.Statement, &c.Importance, &c.Critical, &scope, &required, &c.State, &c.Justification, &support, &c.CreatedAt); err != nil {
			rows.Close()
			return g, err
		}
		c.DecisionID = decision.ID
		_ = json.Unmarshal(scope, &c.Scope)
		_ = json.Unmarshal(required, &c.RequiredEvidenceTypes)
		_ = json.Unmarshal(support, &c.Support)
		g.Claims = append(g.Claims, c)
	}
	rows.Close()
	rows, err = p.pool.Query(ctx, `SELECT id,evidence_type,source,summary,COALESCE(uri,''),content_hash,freshness_at,metadata_json,raw FROM evidence WHERE run_id=$1 ORDER BY id`, run.ID)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var e domain.Evidence
		var metadata, raw []byte
		if err = rows.Scan(&e.ID, &e.Kind, &e.Source, &e.Summary, &e.URI, &e.ContentHash, &e.ObservedAt, &metadata, &raw); err != nil {
			rows.Close()
			return g, err
		}
		e.RunID = run.ID
		e.Metadata = metadata
		e.Raw = raw
		g.Evidence = append(g.Evidence, e)
	}
	rows.Close()
	rows, err = p.pool.Query(ctx, `SELECT r.id,r.from_id,r.to_id,r.relation_type,r.rationale FROM relations r WHERE r.from_id IN (SELECT id FROM claims WHERE decision_id=$1 UNION SELECT id FROM evidence WHERE run_id=$2 UNION SELECT id FROM verifications WHERE decision_id=$1) OR r.to_id IN (SELECT id FROM claims WHERE decision_id=$1 UNION SELECT id FROM evidence WHERE run_id=$2 UNION SELECT id FROM verifications WHERE decision_id=$1) ORDER BY r.id`, decision.ID, run.ID)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var r domain.Relation
		if err = rows.Scan(&r.ID, &r.FromID, &r.ToID, &r.Type, &r.Rationale); err != nil {
			rows.Close()
			return g, err
		}
		g.Relations = append(g.Relations, r)
	}
	rows.Close()
	rows, err = p.pool.Query(ctx, `SELECT id,COALESCE(claim_id,''),statement,critical,status,impact FROM assumptions WHERE decision_id=$1 ORDER BY id`, decision.ID)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var a domain.Assumption
		if err = rows.Scan(&a.ID, &a.ClaimID, &a.Statement, &a.Critical, &a.Status, &a.Impact); err != nil {
			rows.Close()
			return g, err
		}
		a.DecisionID = decision.ID
		g.Assumptions = append(g.Assumptions, a)
	}
	rows.Close()
	rows, err = p.pool.Query(ctx, `SELECT id,question,critical,resolved FROM unknowns WHERE decision_id=$1 ORDER BY id`, decision.ID)
	if err != nil {
		return g, err
	}
	for rows.Next() {
		var u domain.Unknown
		if err = rows.Scan(&u.ID, &u.Question, &u.Critical, &u.Resolved); err != nil {
			rows.Close()
			return g, err
		}
		u.DecisionID = decision.ID
		g.Unknowns = append(g.Unknowns, u)
	}
	rows.Close()
	verifications, err := p.getVerifications(ctx, decision.ID)
	if err != nil {
		return g, err
	}
	g.Verifications = verifications
	evidenceIDs := map[string]bool{}
	for _, e := range g.Evidence {
		evidenceIDs[e.ID] = true
	}
	claimIndex := map[string]int{}
	for i, c := range g.Claims {
		claimIndex[c.ID] = i
		g.Decision.ClaimIDs = append(g.Decision.ClaimIDs, c.ID)
	}
	for _, r := range g.Relations {
		if evidenceIDs[r.FromID] {
			if i, ok := claimIndex[r.ToID]; ok {
				g.Claims[i].EvidenceIDs = append(g.Claims[i].EvidenceIDs, r.FromID)
			}
		}
	}
	return g, nil
}

func (p *Postgres) SaveVerifications(ctx context.Context, values []domain.Verification) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for _, v := range values {
		_, err = tx.Exec(ctx, `INSERT INTO verifications(id,decision_id,claim_id,kind,check_description,specification_json,environment,status,requires_approval,approved,approved_by,outcome,result_json,artifact_hash,created_at,executed_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,NULLIF($11,''),NULLIF($12,''),$13,NULLIF($14,''),$15,$16) ON CONFLICT DO NOTHING`, v.ID, v.DecisionID, v.ClaimID, v.Kind, v.Check, v.Specification, v.Environment, v.Status, v.RequiresApproval, v.Approved, v.ApprovedBy, v.Outcome, nullableJSON(v.Artifact), v.ArtifactHash, v.CreatedAt, v.ExecutedAt)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
func (p *Postgres) getVerifications(ctx context.Context, decisionID string) ([]domain.Verification, error) {
	rows, err := p.pool.Query(ctx, `SELECT id,claim_id,kind,check_description,specification_json,environment,status,requires_approval,approved,COALESCE(approved_by,''),COALESCE(outcome,''),result_json,COALESCE(artifact_hash,''),created_at,executed_at FROM verifications WHERE decision_id=$1 ORDER BY id`, decisionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var values []domain.Verification
	for rows.Next() {
		var v domain.Verification
		var spec, result []byte
		if err = rows.Scan(&v.ID, &v.ClaimID, &v.Kind, &v.Check, &spec, &v.Environment, &v.Status, &v.RequiresApproval, &v.Approved, &v.ApprovedBy, &v.Outcome, &result, &v.ArtifactHash, &v.CreatedAt, &v.ExecutedAt); err != nil {
			return nil, err
		}
		v.DecisionID = decisionID
		v.Specification = spec
		v.Artifact = result
		values = append(values, v)
	}
	return values, rows.Err()
}
func (p *Postgres) GetVerification(ctx context.Context, id string) (domain.Verification, error) {
	var v domain.Verification
	var spec, result []byte
	err := p.pool.QueryRow(ctx, `SELECT decision_id,claim_id,kind,check_description,specification_json,environment,status,requires_approval,approved,COALESCE(approved_by,''),COALESCE(outcome,''),result_json,COALESCE(artifact_hash,''),created_at,executed_at FROM verifications WHERE id=$1`, id).Scan(&v.DecisionID, &v.ClaimID, &v.Kind, &v.Check, &spec, &v.Environment, &v.Status, &v.RequiresApproval, &v.Approved, &v.ApprovedBy, &v.Outcome, &result, &v.ArtifactHash, &v.CreatedAt, &v.ExecutedAt)
	v.ID = id
	v.Specification = spec
	v.Artifact = result
	return v, mapNotFound(err)
}
func (p *Postgres) UpdateVerification(ctx context.Context, v domain.Verification) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `UPDATE verifications SET environment=$2,status=$3,approved=$4,approved_by=NULLIF($5,''),outcome=NULLIF($6,''),result_json=$7,artifact_hash=NULLIF($8,''),executed_at=$9 WHERE id=$1`, v.ID, v.Environment, v.Status, v.Approved, v.ApprovedBy, v.Outcome, nullableJSON(v.Artifact), v.ArtifactHash, v.ExecutedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if v.Outcome != "" {
		state := domain.ClaimContradicted
		if v.Outcome == "passed" {
			state = domain.ClaimExternallyVerified
		}
		_, err = tx.Exec(ctx, `UPDATE claims SET state=$2,support_score=jsonb_set(jsonb_set(support_score,'{direct_verification_strength}',to_jsonb(CASE WHEN $2='externally_verified' THEN 1.0 ELSE 0.0 END)), '{contradiction_burden}',to_jsonb(CASE WHEN $2='contradicted' THEN 1.0 ELSE 0.0 END)) WHERE id=$1`, v.ClaimID, state)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `INSERT INTO relations(id,from_id,to_id,relation_type,rationale) VALUES($1,$2,$3,'verified_by',$4) ON CONFLICT DO NOTHING`, "rel_verified_"+v.ID, v.ClaimID, v.ID, "Controlled verification "+v.Outcome+"; artifact "+v.ArtifactHash)
		if err != nil {
			return err
		}
		if v.Outcome == "passed" {
			_, _ = tx.Exec(ctx, `UPDATE unknowns SET resolved=true WHERE decision_id=$1`, v.DecisionID)
		}
	}
	return tx.Commit(ctx)
}
func (p *Postgres) SaveDecision(ctx context.Context, d domain.Decision) error {
	conditions, _ := json.Marshal(d.Conditions)
	tag, err := p.pool.Exec(ctx, `UPDATE decisions SET verdict=$2,action_allowed=$3,human_approved=$4,conditions=$5,evaluated_at=$6 WHERE id=$1`, d.ID, d.Verdict, d.ActionAllowed, d.HumanApproved, conditions, d.EvaluatedAt)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
func (p *Postgres) GetCertificate(ctx context.Context, id string) (domain.Certificate, error) {
	var data []byte
	err := p.pool.QueryRow(ctx, `SELECT certificate_json FROM proofs WHERE decision_id=$1`, id).Scan(&data)
	if err != nil {
		return domain.Certificate{}, mapNotFound(err)
	}
	var certificate domain.Certificate
	err = json.Unmarshal(data, &certificate)
	return certificate, err
}
func (p *Postgres) SaveCertificate(ctx context.Context, c domain.Certificate) error {
	data, err := json.Marshal(c)
	if err != nil {
		return err
	}
	_, err = p.pool.Exec(ctx, `INSERT INTO proofs(id,decision_id,certificate_json,certificate_hash,created_at) VALUES($1,$2,$3,$4,$5) ON CONFLICT(decision_id) DO NOTHING`, `proof_`+c.DecisionID, c.DecisionID, data, c.Proof.Digest, c.IssuedAt)
	return err
}

func nullableJSON(value json.RawMessage) any {
	if len(value) == 0 {
		return nil
	}
	return value
}
func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

var _ Repository = (*Postgres)(nil)
