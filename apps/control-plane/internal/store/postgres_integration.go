package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

func (p *Postgres) CreateProjectConnection(ctx context.Context, connection domain.ProjectConnection) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO project_connections(id,account_id,project_id,provider,repository,status,token_hash,token_prefix,created_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, connection.ID, connection.AccountID, connection.ProjectID, connection.Provider, connection.Repository, connection.Status, connection.TokenHash, connection.TokenPrefix, connection.CreatedAt)
	return mapPortfolioError(err)
}

func (p *Postgres) GetProjectConnectionByTokenHash(ctx context.Context, tokenHash string) (domain.ProjectConnection, error) {
	var connection domain.ProjectConnection
	err := p.pool.QueryRow(ctx, `SELECT id,account_id,project_id,provider,repository,status,token_hash,token_prefix,created_at,last_seen_at FROM project_connections WHERE token_hash=$1 AND status='active'`, tokenHash).Scan(&connection.ID, &connection.AccountID, &connection.ProjectID, &connection.Provider, &connection.Repository, &connection.Status, &connection.TokenHash, &connection.TokenPrefix, &connection.CreatedAt, &connection.LastSeenAt)
	return connection, mapNotFound(err)
}

func (p *Postgres) RevokeProjectConnection(ctx context.Context, id string, _ time.Time) (domain.ProjectConnection, error) {
	var connection domain.ProjectConnection
	err := p.pool.QueryRow(ctx, `UPDATE project_connections SET status='revoked' WHERE id=$1 RETURNING id,account_id,project_id,provider,repository,status,token_hash,token_prefix,created_at,last_seen_at`, id).Scan(&connection.ID, &connection.AccountID, &connection.ProjectID, &connection.Provider, &connection.Repository, &connection.Status, &connection.TokenHash, &connection.TokenPrefix, &connection.CreatedAt, &connection.LastSeenAt)
	return connection, mapNotFound(err)
}

func (p *Postgres) SaveProjectIngest(ctx context.Context, ingest domain.ProjectIngest) (domain.ProjectIngest, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.ProjectIngest{}, err
	}
	defer tx.Rollback(ctx)
	seenAt := ingest.Connection.CreatedAt
	if ingest.Report != nil {
		report := ingest.Report
		seenAt = report.ReceivedAt
		_, err = tx.Exec(ctx, `INSERT INTO project_reports(id,external_id,account_id,project_id,connection_id,ai_system_id,tool,status,exit_code,summary,repository,commit_sha,branch,workflow,run_url,details_json,received_at) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),$7,$8,$9,$10,NULLIF($11,''),NULLIF($12,''),NULLIF($13,''),NULLIF($14,''),NULLIF($15,''),$16,$17) ON CONFLICT(connection_id,external_id) DO NOTHING`, report.ID, report.ExternalID, report.AccountID, report.ProjectID, report.ConnectionID, report.AISystemID, report.Tool, report.Status, report.ExitCode, report.Summary, report.Repository, report.CommitSHA, report.Branch, report.Workflow, report.RunURL, nullableJSON(report.Details), report.ReceivedAt)
		if err != nil {
			return domain.ProjectIngest{}, err
		}
		if err = tx.QueryRow(ctx, `SELECT id FROM project_reports WHERE connection_id=$1 AND external_id=$2`, report.ConnectionID, report.ExternalID).Scan(&report.ID); err != nil {
			return domain.ProjectIngest{}, err
		}
	}
	if ingest.Certificate != nil {
		certificate := ingest.Certificate
		if ingest.Report == nil {
			seenAt = certificate.ReceivedAt
		}
		artifactHashes, _ := json.Marshal(certificate.ArtifactHashes)
		_, err = tx.Exec(ctx, `INSERT INTO published_certificates(id,external_id,account_id,project_id,connection_id,ai_system_id,decision_id,run_id,status,action_allowed,policy_version,digest,artifact_hashes,issued_at,received_at,certificate_json) VALUES($1,$2,$3,$4,$5,NULLIF($6,''),$7,NULLIF($8,''),$9,$10,$11,$12,$13,$14,$15,$16) ON CONFLICT(project_id,digest) DO NOTHING`, certificate.ID, certificate.ExternalID, certificate.AccountID, certificate.ProjectID, certificate.ConnectionID, certificate.AISystemID, certificate.DecisionID, certificate.RunID, certificate.Status, certificate.ActionAllowed, certificate.PolicyVersion, certificate.Digest, artifactHashes, certificate.IssuedAt, certificate.ReceivedAt, certificate.Raw)
		if err != nil {
			return domain.ProjectIngest{}, err
		}
		if err = tx.QueryRow(ctx, `SELECT id FROM published_certificates WHERE project_id=$1 AND digest=$2`, certificate.ProjectID, certificate.Digest).Scan(&certificate.ID); err != nil {
			return domain.ProjectIngest{}, err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE project_connections SET last_seen_at=$2 WHERE id=$1 AND status='active'`, ingest.Connection.ID, seenAt); err != nil {
		return domain.ProjectIngest{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.ProjectIngest{}, err
	}
	ingest.Connection.LastSeenAt = &seenAt
	return ingest, nil
}
