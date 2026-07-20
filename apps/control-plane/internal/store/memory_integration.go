package store

import (
	"context"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

func (m *Memory) CreateProjectConnection(_ context.Context, connection domain.ProjectConnection) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	project, ok := m.projects[connection.ProjectID]
	if !ok || project.AccountID != connection.AccountID {
		return ErrNotFound
	}
	for _, existing := range m.connections {
		if existing.ProjectID == connection.ProjectID && existing.Status == "active" {
			return ErrConflict
		}
	}
	m.connections[connection.ID] = connection
	return nil
}

func (m *Memory) GetProjectConnectionByTokenHash(_ context.Context, tokenHash string) (domain.ProjectConnection, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, connection := range m.connections {
		if connection.TokenHash == tokenHash && connection.Status == "active" {
			return connection, nil
		}
	}
	return domain.ProjectConnection{}, ErrNotFound
}

func (m *Memory) RevokeProjectConnection(_ context.Context, id string, _ time.Time) (domain.ProjectConnection, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	connection, ok := m.connections[id]
	if !ok {
		return connection, ErrNotFound
	}
	connection.Status = "revoked"
	m.connections[id] = connection
	return connection, nil
}

func (m *Memory) SaveProjectIngest(_ context.Context, ingest domain.ProjectIngest) (domain.ProjectIngest, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	connection, ok := m.connections[ingest.Connection.ID]
	if !ok || connection.Status != "active" {
		return domain.ProjectIngest{}, ErrNotFound
	}
	now := time.Now().UTC()
	if ingest.Report != nil {
		now = ingest.Report.ReceivedAt
		duplicate := false
		for _, existing := range m.reports {
			if existing.ConnectionID == connection.ID && existing.ExternalID == ingest.Report.ExternalID {
				*ingest.Report = existing
				duplicate = true
				break
			}
		}
		if !duplicate {
			m.reports[ingest.Report.ID] = *ingest.Report
		}
	}
	if ingest.Certificate != nil {
		if ingest.Report == nil {
			now = ingest.Certificate.ReceivedAt
		}
		duplicate := false
		for _, existing := range m.publications {
			if existing.ProjectID == connection.ProjectID && existing.Digest == ingest.Certificate.Digest {
				*ingest.Certificate = existing
				duplicate = true
				break
			}
		}
		if !duplicate {
			m.publications[ingest.Certificate.ID] = *ingest.Certificate
		}
	}
	connection.LastSeenAt = &now
	m.connections[connection.ID] = connection
	ingest.Connection = connection
	return ingest, nil
}
