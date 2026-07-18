package file

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

type Provider struct {
	Path   string
	mu     sync.Mutex
	file   *os.File
	closed bool
}

func New(path string) *Provider { return &Provider{Path: path} }

func (p *Provider) Emit(_ context.Context, event epistemic.Event) error {
	return p.append("event", event)
}

func (p *Provider) Evaluate(_ context.Context, request epistemic.DecisionRequest) (epistemic.DecisionResult, error) {
	if err := p.append("decision_request", request); err != nil {
		return epistemic.DecisionResult{}, err
	}
	decisionID := request.DecisionID
	if decisionID == "" {
		decisionID = "offline"
	}
	result := epistemic.DecisionResult{SpecVersion: epistemic.Version, DecisionID: decisionID, Status: "indeterminate", ActionAllowed: false, Reasons: []epistemic.Reason{{Code: "provider_offline", Message: "File provider records requests but does not evaluate them."}}, Conditions: []string{}, EvaluatedAt: time.Now().UTC(), Context: request.Context}
	return result, p.append("decision_result", result)
}

func (p *Provider) Flush(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.file == nil {
		return nil
	}
	return p.file.Sync()
}

func (p *Provider) Shutdown(context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil
	}
	p.closed = true
	if p.file == nil {
		return nil
	}
	err := p.file.Sync()
	if closeErr := p.file.Close(); err == nil {
		err = closeErr
	}
	return err
}

func (p *Provider) append(kind string, value any) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return errors.New("file provider is shut down")
	}
	if p.file == nil {
		if err := os.MkdirAll(filepath.Dir(p.Path), 0o755); err != nil {
			return err
		}
		file, err := os.OpenFile(p.Path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return err
		}
		p.file = file
	}
	line, err := json.Marshal(map[string]any{"kind": kind, "value": value})
	if err != nil {
		return err
	}
	line = append(line, '\n')
	_, err = p.file.Write(line)
	return err
}

var _ epistemic.Provider = (*Provider)(nil)
