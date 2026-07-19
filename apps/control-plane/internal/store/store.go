package store

import (
	"context"
	"errors"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
)

type Repository interface {
	CreateAccount(context.Context, domain.Account) error
	GetAccount(context.Context, string) (domain.Account, error)
	CreateProject(context.Context, domain.Project) error
	GetProject(context.Context, string) (domain.Project, error)
	CreateAISystem(context.Context, domain.AISystem) error
	GetAISystem(context.Context, string) (domain.AISystem, error)
	GetAccountDashboard(context.Context, string, time.Time) (domain.AccountDashboard, error)
	CreateProjectConnection(context.Context, domain.ProjectConnection) error
	RevokeProjectConnection(context.Context, string, time.Time) (domain.ProjectConnection, error)
	GetProjectConnectionByTokenHash(context.Context, string) (domain.ProjectConnection, error)
	SaveProjectIngest(context.Context, domain.ProjectIngest) (domain.ProjectIngest, error)
	CreateRun(context.Context, domain.Run, domain.Decision) error
	GetRun(context.Context, string) (domain.Run, error)
	AddEvent(context.Context, string, domain.Event) (domain.Event, bool, error)
	SaveAnalysis(context.Context, string, domain.Analysis) error
	GetGraphByRun(context.Context, string) (domain.Graph, error)
	GetGraphByDecision(context.Context, string) (domain.Graph, error)
	SaveVerifications(context.Context, []domain.Verification) error
	GetVerification(context.Context, string) (domain.Verification, error)
	UpdateVerification(context.Context, domain.Verification) error
	SaveDecision(context.Context, domain.Decision) error
	GetCertificate(context.Context, string) (domain.Certificate, error)
	SaveCertificate(context.Context, domain.Certificate) error
}
