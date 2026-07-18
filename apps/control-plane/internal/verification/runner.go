package verification

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

var ErrExecutionDisabled = errors.New("verification execution is disabled; submit a recorded sandbox artifact or enable the Docker runner")

type Result struct {
	Outcome  string
	Artifact json.RawMessage
}

type Runner interface {
	Execute(context.Context, domain.Verification) (Result, error)
}

type DisabledRunner struct{}

func (DisabledRunner) Execute(context.Context, domain.Verification) (Result, error) {
	return Result{}, ErrExecutionDisabled
}

type Specification struct {
	Image          string   `json:"image"`
	Repository     string   `json:"repository"`
	Command        []string `json:"command"`
	TimeoutSeconds int      `json:"timeout_seconds"`
}

type DockerRunner struct {
	Root          string
	AllowedImages map[string]bool
}

func NewDockerRunner(root string, images ...string) *DockerRunner {
	allowed := map[string]bool{}
	for _, image := range images {
		allowed[image] = true
	}
	return &DockerRunner{Root: root, AllowedImages: allowed}
}

func (r *DockerRunner) Execute(ctx context.Context, verification domain.Verification) (Result, error) {
	var specification Specification
	if err := json.Unmarshal(verification.Specification, &specification); err != nil {
		return Result{}, fmt.Errorf("decode verification specification: %w", err)
	}
	if !r.AllowedImages[specification.Image] {
		return Result{}, fmt.Errorf("image %q is not in the verification allowlist", specification.Image)
	}
	if len(specification.Command) == 0 {
		return Result{}, errors.New("verification command is empty")
	}
	root, err := filepath.Abs(r.Root)
	if err != nil {
		return Result{}, err
	}
	repository, err := filepath.Abs(filepath.Join(root, filepath.Clean(specification.Repository)))
	if err != nil {
		return Result{}, err
	}
	if repository != root && !strings.HasPrefix(repository, root+string(filepath.Separator)) {
		return Result{}, errors.New("verification repository escapes configured root")
	}
	if specification.TimeoutSeconds <= 0 || specification.TimeoutSeconds > 120 {
		specification.TimeoutSeconds = 60
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(specification.TimeoutSeconds)*time.Second)
	defer cancel()

	arguments := []string{
		"run", "--rm", "--network", "none", "--cpus", "1", "--memory", "512m",
		"--pids-limit", "128", "--read-only", "--tmpfs", "/tmp:rw,noexec,nosuid,size=64m",
		"-e", "PYTHONDONTWRITEBYTECODE=1", "-v", repository + ":/workspace:ro", "-w", "/workspace",
		specification.Image,
	}
	arguments = append(arguments, specification.Command...)
	command := exec.CommandContext(runCtx, "docker", arguments...)
	var stdout, stderr limitedBuffer
	command.Stdout, command.Stderr = &stdout, &stderr
	started := time.Now().UTC()
	err = command.Run()
	duration := time.Since(started)
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			exitCode = exitError.ExitCode()
		} else if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
			exitCode = 124
		} else {
			return Result{}, fmt.Errorf("start Docker verification: %w", err)
		}
	}
	outcome := "passed"
	if exitCode != 0 {
		outcome = "failed"
	}
	artifact, err := json.Marshal(map[string]any{
		"runner": "docker", "image": specification.Image, "command": specification.Command,
		"exit_code": exitCode, "stdout": stdout.String(), "stderr": stderr.String(),
		"duration_ms": duration.Milliseconds(), "network": "none",
	})
	return Result{Outcome: outcome, Artifact: artifact}, err
}

type limitedBuffer struct{ bytes.Buffer }

func (b *limitedBuffer) Write(p []byte) (int, error) {
	const limit = 1 << 20
	remaining := limit - b.Len()
	if remaining <= 0 {
		return len(p), nil
	}
	if len(p) > remaining {
		_, _ = b.Buffer.Write(p[:remaining])
		return len(p), nil
	}
	return b.Buffer.Write(p)
}
