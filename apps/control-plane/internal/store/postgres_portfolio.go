package store

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func (p *Postgres) CreateAccount(ctx context.Context, account domain.Account) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO accounts(id,name,slug,created_at) VALUES($1,$2,$3,$4)`, account.ID, account.Name, account.Slug, account.CreatedAt)
	return mapPortfolioError(err)
}

func (p *Postgres) GetAccount(ctx context.Context, id string) (domain.Account, error) {
	var account domain.Account
	err := p.pool.QueryRow(ctx, `SELECT id,name,slug,created_at FROM accounts WHERE id=$1`, id).Scan(&account.ID, &account.Name, &account.Slug, &account.CreatedAt)
	return account, mapNotFound(err)
}

func (p *Postgres) CreateProject(ctx context.Context, project domain.Project) error {
	_, err := p.pool.Exec(ctx, `INSERT INTO projects(id,account_id,name,slug,repository,owner,created_at) VALUES($1,$2,$3,$4,NULLIF($5,''),NULLIF($6,''),$7)`, project.ID, project.AccountID, project.Name, project.Slug, project.Repository, project.Owner, project.CreatedAt)
	return mapPortfolioError(err)
}

func (p *Postgres) GetProject(ctx context.Context, id string) (domain.Project, error) {
	var project domain.Project
	err := p.pool.QueryRow(ctx, `SELECT id,account_id,name,slug,COALESCE(repository,''),COALESCE(owner,''),created_at FROM projects WHERE id=$1`, id).Scan(&project.ID, &project.AccountID, &project.Name, &project.Slug, &project.Repository, &project.Owner, &project.CreatedAt)
	return project, mapNotFound(err)
}

func (p *Postgres) CreateAISystem(ctx context.Context, system domain.AISystem) error {
	dataClasses, _ := json.Marshal(system.DataClasses)
	tools, _ := json.Marshal(system.Tools)
	_, err := p.pool.Exec(ctx, `INSERT INTO ai_systems(id,account_id,project_id,name,provider,model,purpose,data_classes,tools,owner,status,created_at,updated_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,NULLIF($10,''),$11,$12,$13)`, system.ID, system.AccountID, system.ProjectID, system.Name, system.Provider, system.Model, system.Purpose, dataClasses, tools, system.Owner, system.Status, system.CreatedAt, system.UpdatedAt)
	return mapPortfolioError(err)
}

func (p *Postgres) GetAISystem(ctx context.Context, id string) (domain.AISystem, error) {
	var system domain.AISystem
	var dataClasses, tools []byte
	err := p.pool.QueryRow(ctx, `SELECT id,account_id,project_id,name,provider,model,purpose,data_classes,tools,COALESCE(owner,''),status,created_at,updated_at,last_used_at FROM ai_systems WHERE id=$1`, id).Scan(&system.ID, &system.AccountID, &system.ProjectID, &system.Name, &system.Provider, &system.Model, &system.Purpose, &dataClasses, &tools, &system.Owner, &system.Status, &system.CreatedAt, &system.UpdatedAt, &system.LastUsedAt)
	if err != nil {
		return system, mapNotFound(err)
	}
	_ = json.Unmarshal(dataClasses, &system.DataClasses)
	_ = json.Unmarshal(tools, &system.Tools)
	return system, nil
}

func (p *Postgres) GetAccountDashboard(ctx context.Context, accountID string, generatedAt time.Time) (domain.AccountDashboard, error) {
	account, err := p.GetAccount(ctx, accountID)
	if err != nil {
		return domain.AccountDashboard{}, err
	}
	snapshot := NewMemory()
	snapshot.accounts[account.ID] = account

	projectRows, err := p.pool.Query(ctx, `SELECT id,account_id,name,slug,COALESCE(repository,''),COALESCE(owner,''),created_at FROM projects WHERE account_id=$1 ORDER BY name`, accountID)
	if err != nil {
		return domain.AccountDashboard{}, err
	}
	for projectRows.Next() {
		var project domain.Project
		if err = projectRows.Scan(&project.ID, &project.AccountID, &project.Name, &project.Slug, &project.Repository, &project.Owner, &project.CreatedAt); err != nil {
			projectRows.Close()
			return domain.AccountDashboard{}, err
		}
		snapshot.projects[project.ID] = project
	}
	if err = projectRows.Err(); err != nil {
		projectRows.Close()
		return domain.AccountDashboard{}, err
	}
	projectRows.Close()

	systemRows, err := p.pool.Query(ctx, `SELECT id,account_id,project_id,name,provider,model,purpose,data_classes,tools,COALESCE(owner,''),status,created_at,updated_at,last_used_at FROM ai_systems WHERE account_id=$1 ORDER BY name`, accountID)
	if err != nil {
		return domain.AccountDashboard{}, err
	}
	for systemRows.Next() {
		var system domain.AISystem
		var dataClasses, tools []byte
		if err = systemRows.Scan(&system.ID, &system.AccountID, &system.ProjectID, &system.Name, &system.Provider, &system.Model, &system.Purpose, &dataClasses, &tools, &system.Owner, &system.Status, &system.CreatedAt, &system.UpdatedAt, &system.LastUsedAt); err != nil {
			systemRows.Close()
			return domain.AccountDashboard{}, err
		}
		_ = json.Unmarshal(dataClasses, &system.DataClasses)
		_ = json.Unmarshal(tools, &system.Tools)
		snapshot.aiSystems[system.ID] = system
	}
	if err = systemRows.Err(); err != nil {
		systemRows.Close()
		return domain.AccountDashboard{}, err
	}
	systemRows.Close()

	connectionRows, err := p.pool.Query(ctx, `SELECT id,account_id,project_id,provider,repository,status,token_hash,token_prefix,created_at,last_seen_at FROM project_connections WHERE account_id=$1 ORDER BY created_at`, accountID)
	if err != nil {
		return domain.AccountDashboard{}, err
	}
	for connectionRows.Next() {
		var connection domain.ProjectConnection
		if err = connectionRows.Scan(&connection.ID, &connection.AccountID, &connection.ProjectID, &connection.Provider, &connection.Repository, &connection.Status, &connection.TokenHash, &connection.TokenPrefix, &connection.CreatedAt, &connection.LastSeenAt); err != nil {
			connectionRows.Close()
			return domain.AccountDashboard{}, err
		}
		snapshot.connections[connection.ID] = connection
	}
	if err = connectionRows.Err(); err != nil {
		connectionRows.Close()
		return domain.AccountDashboard{}, err
	}
	connectionRows.Close()

	reportRows, err := p.pool.Query(ctx, `SELECT id,external_id,account_id,project_id,connection_id,COALESCE(ai_system_id,''),tool,status,exit_code,summary,COALESCE(repository,''),COALESCE(commit_sha,''),COALESCE(branch,''),COALESCE(workflow,''),COALESCE(run_url,''),details_json,received_at FROM project_reports WHERE account_id=$1 ORDER BY received_at DESC`, accountID)
	if err != nil {
		return domain.AccountDashboard{}, err
	}
	for reportRows.Next() {
		var report domain.ProjectReport
		if err = reportRows.Scan(&report.ID, &report.ExternalID, &report.AccountID, &report.ProjectID, &report.ConnectionID, &report.AISystemID, &report.Tool, &report.Status, &report.ExitCode, &report.Summary, &report.Repository, &report.CommitSHA, &report.Branch, &report.Workflow, &report.RunURL, &report.Details, &report.ReceivedAt); err != nil {
			reportRows.Close()
			return domain.AccountDashboard{}, err
		}
		snapshot.reports[report.ID] = report
	}
	if err = reportRows.Err(); err != nil {
		reportRows.Close()
		return domain.AccountDashboard{}, err
	}
	reportRows.Close()

	publicationRows, err := p.pool.Query(ctx, `SELECT id,external_id,account_id,project_id,connection_id,COALESCE(ai_system_id,''),decision_id,COALESCE(run_id,''),status,action_allowed,policy_version,digest,artifact_hashes,issued_at,received_at,certificate_json FROM published_certificates WHERE account_id=$1 ORDER BY issued_at DESC`, accountID)
	if err != nil {
		return domain.AccountDashboard{}, err
	}
	for publicationRows.Next() {
		var publication domain.PublishedCertificate
		var artifactHashes []byte
		if err = publicationRows.Scan(&publication.ID, &publication.ExternalID, &publication.AccountID, &publication.ProjectID, &publication.ConnectionID, &publication.AISystemID, &publication.DecisionID, &publication.RunID, &publication.Status, &publication.ActionAllowed, &publication.PolicyVersion, &publication.Digest, &artifactHashes, &publication.IssuedAt, &publication.ReceivedAt, &publication.Raw); err != nil {
			publicationRows.Close()
			return domain.AccountDashboard{}, err
		}
		_ = json.Unmarshal(artifactHashes, &publication.ArtifactHashes)
		snapshot.publications[publication.ID] = publication
	}
	if err = publicationRows.Err(); err != nil {
		publicationRows.Close()
		return domain.AccountDashboard{}, err
	}
	publicationRows.Close()

	runRows, err := p.pool.Query(ctx, `SELECT id FROM runs WHERE account_id=$1 ORDER BY created_at DESC`, accountID)
	if err != nil {
		return domain.AccountDashboard{}, err
	}
	var runIDs []string
	for runRows.Next() {
		var runID string
		if err = runRows.Scan(&runID); err != nil {
			runRows.Close()
			return domain.AccountDashboard{}, err
		}
		runIDs = append(runIDs, runID)
	}
	if err = runRows.Err(); err != nil {
		runRows.Close()
		return domain.AccountDashboard{}, err
	}
	runRows.Close()

	for _, runID := range runIDs {
		graph, graphErr := p.GetGraphByRun(ctx, runID)
		if graphErr != nil {
			return domain.AccountDashboard{}, graphErr
		}
		snapshot.runs[graph.Run.ID] = graph.Run
		snapshot.decisions[graph.Decision.ID] = graph.Decision
		for _, value := range graph.Claims {
			snapshot.claims[value.ID] = value
		}
		for _, value := range graph.Evidence {
			snapshot.evidence[value.ID] = value
		}
		for _, value := range graph.Relations {
			snapshot.relations[value.ID] = value
		}
		for _, value := range graph.Assumptions {
			snapshot.assumptions[value.ID] = value
		}
		for _, value := range graph.Unknowns {
			snapshot.unknowns[value.ID] = value
		}
		for _, value := range graph.Verifications {
			snapshot.verifications[value.ID] = value
		}
		if certificate, certificateErr := p.GetCertificate(ctx, graph.Decision.ID); certificateErr == nil {
			snapshot.certificates[graph.Decision.ID] = certificate
		} else if !errors.Is(certificateErr, ErrNotFound) {
			return domain.AccountDashboard{}, certificateErr
		}
	}
	return snapshot.GetAccountDashboard(ctx, accountID, generatedAt)
}

func mapPortfolioError(err error) error {
	if err == nil {
		return nil
	}
	var postgresError *pgconn.PgError
	if errors.As(err, &postgresError) {
		switch postgresError.Code {
		case "23505":
			return ErrConflict
		case "23503":
			return ErrNotFound
		}
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
