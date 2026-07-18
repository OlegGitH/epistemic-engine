package pipeline

import (
	"strings"
	"testing"
)

func TestGenerateGitHubCreatesVendorNeutralEpistemicGate(t *testing.T) {
	output, err := GenerateGitHub(GenerateInput{})
	if err != nil {
		t.Fatal(err)
	}
	if len(output.Files) != 1 || output.Files[0].Path != ".github/workflows/epistemic-ci.yml" {
		t.Fatalf("unexpected generated files: %+v", output.Files)
	}
	content := output.Files[0].Content
	for _, expected := range []string{"Epistemic quality gate", "config: .epistemic.yaml", "actions/checkout@v6"} {
		if !strings.Contains(content, expected) {
			t.Fatalf("workflow does not contain %q", expected)
		}
	}
	if strings.Contains(content, "        run:") {
		t.Fatal("workflow unexpectedly hard-codes a project command")
	}
	if len(output.RequiredSecrets) != 0 || len(output.RequiredVariables) != 0 {
		t.Fatalf("unexpected requirements: secrets=%v variables=%v", output.RequiredSecrets, output.RequiredVariables)
	}
}

func TestGenerateGitHubRejectsEscapingPath(t *testing.T) {
	_, err := GenerateGitHub(GenerateInput{ConfigPath: "../outside.yaml"})
	if err == nil {
		t.Fatal("expected unsafe path to be rejected")
	}
}

func TestGenerateGitHubRejectsEscapingLocalAction(t *testing.T) {
	_, err := GenerateGitHub(GenerateInput{EpistemicAction: "./../../outside"})
	if err == nil {
		t.Fatal("expected unsafe local action path to be rejected")
	}
}
