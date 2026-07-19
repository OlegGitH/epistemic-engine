package store

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

func (m *Memory) CreateAccount(_ context.Context, account domain.Account) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, existing := range m.accounts {
		if existing.Slug == account.Slug {
			return fmt.Errorf("%w: account slug already exists", ErrConflict)
		}
	}
	m.accounts[account.ID] = account
	return nil
}

func (m *Memory) GetAccount(_ context.Context, id string) (domain.Account, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	account, ok := m.accounts[id]
	if !ok {
		return account, ErrNotFound
	}
	return account, nil
}

func (m *Memory) CreateProject(_ context.Context, project domain.Project) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.accounts[project.AccountID]; !ok {
		return ErrNotFound
	}
	for _, existing := range m.projects {
		if existing.AccountID == project.AccountID && existing.Slug == project.Slug {
			return fmt.Errorf("%w: project slug already exists", ErrConflict)
		}
	}
	m.projects[project.ID] = project
	return nil
}

func (m *Memory) GetProject(_ context.Context, id string) (domain.Project, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	project, ok := m.projects[id]
	if !ok {
		return project, ErrNotFound
	}
	return project, nil
}

func (m *Memory) CreateAISystem(_ context.Context, system domain.AISystem) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	project, ok := m.projects[system.ProjectID]
	if !ok || project.AccountID != system.AccountID {
		return ErrNotFound
	}
	m.aiSystems[system.ID] = system
	return nil
}

func (m *Memory) GetAISystem(_ context.Context, id string) (domain.AISystem, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	system, ok := m.aiSystems[id]
	if !ok {
		return system, ErrNotFound
	}
	return system, nil
}

func (m *Memory) GetAccountDashboard(_ context.Context, accountID string, generatedAt time.Time) (domain.AccountDashboard, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	account, ok := m.accounts[accountID]
	if !ok {
		return domain.AccountDashboard{}, ErrNotFound
	}
	dashboard := domain.AccountDashboard{
		Account: account, GeneratedAt: generatedAt,
		Projects: []domain.ProjectSummary{}, AISystems: []domain.AISystemSummary{},
		Connections: []domain.ConnectionSummary{}, Reports: []domain.ProjectReport{},
		Certificates: []domain.CertificateSummary{}, Activity: []domain.DashboardActivity{},
	}
	projectIndex := map[string]int{}
	for _, project := range m.projects {
		if project.AccountID != accountID {
			continue
		}
		summary := domain.ProjectSummary{Project: project, CertificationStatus: "uncertified", ConnectionStatus: "disconnected"}
		dashboard.Projects = append(dashboard.Projects, summary)
		projectIndex[project.ID] = len(dashboard.Projects) - 1
	}
	connectionIndex := map[string]int{}
	for _, connection := range m.connections {
		if connection.AccountID != accountID {
			continue
		}
		summary := domain.ConnectionSummary{ProjectConnection: connection}
		if index, exists := projectIndex[connection.ProjectID]; exists {
			project := &dashboard.Projects[index]
			if connection.Status == "active" || project.ConnectionStatus == "disconnected" {
				project.ConnectionStatus = connection.Status
			}
			summary.ProjectName = project.Name
		}
		dashboard.Connections = append(dashboard.Connections, summary)
		connectionIndex[connection.ID] = len(dashboard.Connections) - 1
	}
	for _, report := range m.reports {
		if report.AccountID != accountID {
			continue
		}
		dashboard.Reports = append(dashboard.Reports, report)
		if index, exists := projectIndex[report.ProjectID]; exists {
			dashboard.Projects[index].Reports++
		}
		if index, exists := connectionIndex[report.ConnectionID]; exists {
			dashboard.Connections[index].Reports++
		}
		dashboard.Activity = append(dashboard.Activity, domain.DashboardActivity{ID: report.ID, Kind: "report", Title: report.Tool + " report", Detail: report.Summary, Status: report.Status, OccurredAt: report.ReceivedAt})
	}

	systemIndex := map[string]int{}
	for _, system := range m.aiSystems {
		if system.AccountID != accountID {
			continue
		}
		summary := domain.AISystemSummary{AISystem: system, CertificationStatus: "uncertified"}
		if index, exists := projectIndex[system.ProjectID]; exists {
			project := &dashboard.Projects[index]
			summary.ProjectName = project.Name
			project.AISystems++
		}
		dashboard.AISystems = append(dashboard.AISystems, summary)
		systemIndex[system.ID] = len(dashboard.AISystems) - 1
		dashboard.Activity = append(dashboard.Activity, domain.DashboardActivity{ID: system.ID, Kind: "ai_system", Title: system.Name, Detail: system.Provider + " / " + system.Model, Status: system.Status, OccurredAt: system.CreatedAt})
	}

	latestRunByProject := map[string]domain.Run{}
	latestRunBySystem := map[string]domain.Run{}
	for _, run := range m.runs {
		projectPosition, hasProject := projectIndex[run.ProjectID]
		if run.AccountID != accountID && !hasProject {
			continue
		}
		var project *domain.ProjectSummary
		if hasProject {
			project = &dashboard.Projects[projectPosition]
			project.Runs++
			if project.LastActivityAt == nil || run.CreatedAt.After(*project.LastActivityAt) {
				activityAt := run.CreatedAt
				project.LastActivityAt = &activityAt
				project.LatestRunID = run.ID
			}
			if latest, exists := latestRunByProject[project.ID]; !exists || run.CreatedAt.After(latest.CreatedAt) {
				latestRunByProject[project.ID] = run
			}
		}
		if position, exists := systemIndex[run.AISystemID]; exists {
			system := &dashboard.AISystems[position]
			usedAt := run.CreatedAt
			system.LastUsedAt = &usedAt
			if latest, exists := latestRunBySystem[system.ID]; !exists || run.CreatedAt.After(latest.CreatedAt) {
				latestRunBySystem[system.ID] = run
			}
		}
		dashboard.Activity = append(dashboard.Activity, domain.DashboardActivity{ID: run.ID, Kind: "run", Title: run.Title, Detail: run.Recommendation, Status: string(run.Status), OccurredAt: run.CreatedAt, RunID: run.ID})

		decision := m.decisions[run.DecisionID]
		for _, claim := range m.claims {
			if claim.DecisionID != decision.ID {
				continue
			}
			dashboard.Knowledge.Claims++
			if project != nil {
				project.Claims++
			}
			switch claim.State {
			case domain.ClaimSupported, domain.ClaimExternallyVerified:
				dashboard.Knowledge.SupportedClaims++
				if project != nil {
					project.SupportedClaims++
				}
			case domain.ClaimContradicted, domain.ClaimRejected:
				dashboard.Knowledge.ContradictedClaims++
			case domain.ClaimStale:
				dashboard.Knowledge.StaleClaims++
			}
		}
		for _, evidence := range m.evidence {
			if evidence.RunID == run.ID {
				dashboard.Knowledge.EvidenceArtifacts++
				if project != nil {
					project.EvidenceArtifacts++
				}
			}
		}
		for _, unknown := range m.unknowns {
			if unknown.DecisionID == decision.ID && !unknown.Resolved {
				dashboard.Knowledge.OpenUnknowns++
				if project != nil {
					project.OpenUnknowns++
				}
			}
		}

		if certificate, exists := m.certificates[decision.ID]; exists {
			projectName, systemName := "", ""
			if project != nil {
				projectName = project.Name
			}
			if position, exists := systemIndex[run.AISystemID]; exists {
				system := &dashboard.AISystems[position]
				systemName = system.Name
			}
			dashboard.Certificates = append(dashboard.Certificates, domain.CertificateSummary{
				DecisionID: decision.ID, RunID: run.ID, ProjectID: run.ProjectID, ProjectName: projectName,
				AISystemID: run.AISystemID, AISystemName: systemName, Verdict: string(certificate.Verdict),
				ActionAllowed: certificate.ActionAllowed, PolicyVersion: certificate.PolicyVersion,
				IssuedAt: certificate.IssuedAt, Digest: certificate.Proof.Digest, Source: "control-plane", ReceivedAt: certificate.IssuedAt,
			})
			status := certificateStatus(certificate)
			dashboard.Activity = append(dashboard.Activity, domain.DashboardActivity{ID: decision.ID, Kind: "certificate", Title: "Decision certificate issued", Detail: string(certificate.Verdict), Status: status, OccurredAt: certificate.IssuedAt, RunID: run.ID})
		}
	}

	latestPublicationByProject := map[string]domain.PublishedCertificate{}
	latestPublicationBySystem := map[string]domain.PublishedCertificate{}
	for _, publication := range m.publications {
		if publication.AccountID != accountID {
			continue
		}
		projectName, systemName := "", ""
		if index, exists := projectIndex[publication.ProjectID]; exists {
			projectName = dashboard.Projects[index].Name
			if latest, ok := latestPublicationByProject[publication.ProjectID]; !ok || publication.IssuedAt.After(latest.IssuedAt) {
				latestPublicationByProject[publication.ProjectID] = publication
			}
		}
		if index, exists := systemIndex[publication.AISystemID]; exists {
			systemName = dashboard.AISystems[index].Name
			if latest, ok := latestPublicationBySystem[publication.AISystemID]; !ok || publication.IssuedAt.After(latest.IssuedAt) {
				latestPublicationBySystem[publication.AISystemID] = publication
			}
		}
		dashboard.Certificates = append(dashboard.Certificates, domain.CertificateSummary{
			DecisionID: publication.DecisionID, RunID: publication.RunID, ProjectID: publication.ProjectID, ProjectName: projectName,
			AISystemID: publication.AISystemID, AISystemName: systemName, Verdict: publication.Status,
			ActionAllowed: publication.ActionAllowed, PolicyVersion: publication.PolicyVersion,
			IssuedAt: publication.IssuedAt, Digest: publication.Digest, Source: "connected-project", ReceivedAt: publication.ReceivedAt,
		})
		dashboard.Activity = append(dashboard.Activity, domain.DashboardActivity{ID: publication.ID, Kind: "certificate", Title: "Connected project certificate", Detail: publication.Status, Status: publicationStatus(publication), OccurredAt: publication.ReceivedAt, RunID: publication.RunID})
	}

	for index := range dashboard.Projects {
		project := &dashboard.Projects[index]
		project.KnowledgeCoveragePct = percentage(project.SupportedClaims, project.Claims)
		if run, exists := latestRunByProject[project.ID]; exists {
			if certificate, ok := m.certificates[run.DecisionID]; ok {
				project.CertificationStatus = certificateStatus(certificate)
			} else {
				project.CertificationStatus = "pending"
			}
		}
		if publication, exists := latestPublicationByProject[project.ID]; exists {
			project.CertificationStatus = publicationStatus(publication)
			if project.LastActivityAt == nil || publication.ReceivedAt.After(*project.LastActivityAt) {
				activityAt := publication.ReceivedAt
				project.LastActivityAt = &activityAt
			}
		}
	}
	for index := range dashboard.AISystems {
		system := &dashboard.AISystems[index]
		if run, exists := latestRunBySystem[system.ID]; exists {
			if certificate, ok := m.certificates[run.DecisionID]; ok {
				system.CertificationStatus = certificateStatus(certificate)
				system.CertificateDigest = certificate.Proof.Digest
				evaluatedAt := certificate.IssuedAt
				system.LastEvaluatedAt = &evaluatedAt
			} else {
				system.CertificationStatus = "pending"
			}
		}
		if publication, exists := latestPublicationBySystem[system.ID]; exists {
			system.CertificationStatus = publicationStatus(publication)
			system.CertificateDigest = publication.Digest
			evaluatedAt := publication.IssuedAt
			system.LastEvaluatedAt = &evaluatedAt
		}
	}

	dashboard.Metrics.Projects = len(dashboard.Projects)
	dashboard.Metrics.Reports = len(dashboard.Reports)
	dashboard.Metrics.AISystems = len(dashboard.AISystems)
	for _, project := range dashboard.Projects {
		if project.ConnectionStatus == "active" {
			dashboard.Metrics.ConnectedProjects++
		}
	}
	dashboard.Metrics.KnowledgeCoveragePct = percentage(dashboard.Knowledge.SupportedClaims, dashboard.Knowledge.Claims)
	for _, certificate := range dashboard.Certificates {
		if certificate.ActionAllowed {
			dashboard.Metrics.ValidCertificates++
		}
	}
	dashboard.Metrics.AttentionItems = dashboard.Knowledge.ContradictedClaims + dashboard.Knowledge.OpenUnknowns + dashboard.Knowledge.StaleClaims
	for _, system := range dashboard.AISystems {
		if system.CertificationStatus != "valid" {
			dashboard.Metrics.AttentionItems++
		}
	}

	sort.Slice(dashboard.Projects, func(i, j int) bool { return dashboard.Projects[i].Name < dashboard.Projects[j].Name })
	sort.Slice(dashboard.AISystems, func(i, j int) bool { return dashboard.AISystems[i].Name < dashboard.AISystems[j].Name })
	sort.Slice(dashboard.Certificates, func(i, j int) bool {
		return dashboard.Certificates[i].IssuedAt.After(dashboard.Certificates[j].IssuedAt)
	})
	sort.Slice(dashboard.Reports, func(i, j int) bool { return dashboard.Reports[i].ReceivedAt.After(dashboard.Reports[j].ReceivedAt) })
	sort.Slice(dashboard.Connections, func(i, j int) bool {
		return dashboard.Connections[i].ProjectName < dashboard.Connections[j].ProjectName
	})
	sort.Slice(dashboard.Activity, func(i, j int) bool { return dashboard.Activity[i].OccurredAt.After(dashboard.Activity[j].OccurredAt) })
	if len(dashboard.Activity) > 12 {
		dashboard.Activity = dashboard.Activity[:12]
	}
	return dashboard, nil
}

func publicationStatus(certificate domain.PublishedCertificate) string {
	if certificate.ActionAllowed {
		return "valid"
	}
	return "blocked"
}

func certificateStatus(certificate domain.Certificate) string {
	if certificate.ActionAllowed {
		return "valid"
	}
	return "blocked"
}

func percentage(numerator, denominator int) int {
	if denominator == 0 {
		return 0
	}
	return int(float64(numerator)/float64(denominator)*100 + 0.5)
}
