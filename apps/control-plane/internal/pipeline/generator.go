package pipeline

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"
)

const githubToolID = "github-actions-pipeline"

var (
	safePath  = regexp.MustCompile(`^[A-Za-z0-9._/-]+$`)
	actionRef = regexp.MustCompile(`^[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(?:/[A-Za-z0-9_.-]+)*@[A-Za-z0-9_.-]+$`)
)

type Tool struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Provider     string   `json:"provider"`
	Capabilities []string `json:"capabilities"`
}

type GenerateInput struct {
	Name            string `json:"name"`
	EpistemicAction string `json:"epistemic_action"`
	ConfigPath      string `json:"config_path"`
	CertificatePath string `json:"certificate_path"`
}

type GeneratedFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type GenerateOutput struct {
	ToolID            string          `json:"tool_id"`
	Files             []GeneratedFile `json:"files"`
	RequiredSecrets   []string        `json:"required_secrets"`
	RequiredVariables []string        `json:"required_variables"`
}

func Catalog() []Tool {
	return []Tool{{
		ID: githubToolID, Name: "GitHub Actions pipeline", Provider: "github-actions",
		Description:  "Generate a vendor-neutral GitHub Actions pipeline that evaluates project evidence with Epistemic Engine.",
		Capabilities: []string{"generate", "tool-evidence", "epistemic-quality-gate"},
	}}
}

func GenerateGitHub(input GenerateInput) (GenerateOutput, error) {
	applyDefaults(&input)
	if err := validate(input); err != nil {
		return GenerateOutput{}, err
	}

	var workflow strings.Builder
	fmt.Fprintf(&workflow, "name: %q\n\n", input.Name)
	workflow.WriteString("on:\n  push:\n  pull_request:\n\npermissions:\n  contents: read\n\n")
	workflow.WriteString("concurrency:\n  group: ${{ github.workflow }}-${{ github.ref }}\n  cancel-in-progress: true\n\n")
	workflow.WriteString("jobs:\n  epistemic-gate:\n    name: Epistemic quality gate\n    runs-on: ubuntu-latest\n    timeout-minutes: 10\n    steps:\n")
	workflow.WriteString("      - uses: actions/checkout@v6\n")
	workflow.WriteString("      - name: Evaluate configured evidence\n")
	fmt.Fprintf(&workflow, "        uses: %s\n", input.EpistemicAction)
	workflow.WriteString("        with:\n")
	fmt.Fprintf(&workflow, "          config: %s\n", input.ConfigPath)
	fmt.Fprintf(&workflow, "          certificate: %s\n", input.CertificatePath)

	return GenerateOutput{
		ToolID:          githubToolID,
		Files:           []GeneratedFile{{Path: ".github/workflows/epistemic-ci.yml", Content: workflow.String()}},
		RequiredSecrets: []string{}, RequiredVariables: []string{},
	}, nil
}

func applyDefaults(input *GenerateInput) {
	if input.Name == "" {
		input.Name = "Epistemic CI"
	}
	if input.EpistemicAction == "" {
		input.EpistemicAction = "./adapters/github-action"
	}
	if input.ConfigPath == "" {
		input.ConfigPath = ".epistemic.yaml"
	}
	if input.CertificatePath == "" {
		input.CertificatePath = ".epistemic/certificate.json"
	}
}

func validate(input GenerateInput) error {
	if strings.ContainsAny(input.Name, "\r\n") {
		return errors.New("pipeline name must be one line")
	}
	for _, value := range []string{input.ConfigPath, input.CertificatePath} {
		clean := path.Clean(strings.ReplaceAll(value, "\\", "/"))
		if value == "" || clean != value || strings.HasPrefix(clean, "../") || strings.HasPrefix(clean, "/") || !safePath.MatchString(clean) {
			return fmt.Errorf("unsafe repository-relative path %q", value)
		}
	}
	if strings.ContainsAny(input.EpistemicAction, "\r\n") {
		return errors.New("epistemic_action must be one line")
	}
	if strings.HasPrefix(input.EpistemicAction, "./") {
		local := strings.TrimPrefix(input.EpistemicAction, "./")
		clean := path.Clean(local)
		if local == "" || clean != local || strings.HasPrefix(clean, "../") || !safePath.MatchString(clean) {
			return errors.New("epistemic_action contains an unsafe local path")
		}
	} else if !actionRef.MatchString(input.EpistemicAction) {
		return errors.New("epistemic_action must be a local action path or owner/repository/path@ref")
	}
	return nil
}
