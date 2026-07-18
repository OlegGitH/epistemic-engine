package verification

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

func TestDockerRunnerRejectsUnapprovedImageBeforeExecution(t *testing.T) {
	specification, _ := json.Marshal(Specification{Image: "untrusted:latest", Repository: ".", Command: []string{"true"}})
	runner := NewDockerRunner(t.TempDir(), "python:3.12-alpine")
	_, err := runner.Execute(context.Background(), domain.Verification{Specification: specification})
	if err == nil || !strings.Contains(err.Error(), "allowlist") {
		t.Fatalf("expected allowlist rejection, got %v", err)
	}
}

func TestDockerRunnerRejectsRepositoryEscapeBeforeExecution(t *testing.T) {
	specification, _ := json.Marshal(Specification{Image: "python:3.12-alpine", Repository: "../outside", Command: []string{"true"}})
	runner := NewDockerRunner(t.TempDir(), "python:3.12-alpine")
	_, err := runner.Execute(context.Background(), domain.Verification{Specification: specification})
	if err == nil || !strings.Contains(err.Error(), "escapes") {
		t.Fatalf("expected repository escape rejection, got %v", err)
	}
}
