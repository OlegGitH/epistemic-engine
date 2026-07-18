package store

import (
	"context"
	"errors"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Repository interface {
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
