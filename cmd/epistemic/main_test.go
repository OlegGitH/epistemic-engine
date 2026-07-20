package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

func TestEvaluateStableModesAndCertificate(t *testing.T) {
	directory := t.TempDir()
	junitPath := filepath.Join(directory, "junit.xml")
	if err := os.WriteFile(junitPath, []byte(`<testsuite name="failure" tests="1" failures="1"><testcase name="x"><failure/></testcase></testsuite>`), 0o600); err != nil {
		t.Fatal(err)
	}
	certificatePath := filepath.Join(directory, "certificate.json")
	reportPath := filepath.Join(directory, "certificate-report.md")
	configPath := filepath.Join(directory, ".epistemic.yaml")
	configuration := "api_version: epistemic.dev/v1alpha1\nmode: enforce\ndecision:\n  id: cli-test\n  recommendation: safe\n  action_type: deploy\n  subject_type: repository\n  subject_id: fixture\nprovider:\n  type: local\nsources:\n  - type: junit\n    path: " + filepath.ToSlash(junitPath) + "\noutputs:\n  certificate: " + filepath.ToSlash(certificatePath) + "\n  report: " + filepath.ToSlash(reportPath) + "\n"
	if err := os.WriteFile(configPath, []byte(configuration), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := evaluate([]string{"--config", configPath}); code != 2 {
		t.Fatalf("enforce exit=%d want=2", code)
	}
	data, err := os.ReadFile(certificatePath)
	if err != nil {
		t.Fatal(err)
	}
	var certificate epistemic.Certificate
	if err = json.Unmarshal(data, &certificate); err != nil {
		t.Fatal(err)
	}
	report, err := os.ReadFile(reportPath)
	if err != nil || !strings.Contains(string(report), "Epistemic Decision Report") || !strings.Contains(string(report), certificate.Proof.Digest) {
		t.Fatalf("human report missing or incomplete: err=%v report=%s", err, report)
	}
	expected := certificate.Proof.Digest
	certificate.Proof.Digest = ""
	digest, err := epistemic.Hash(certificate)
	if err != nil || digest != expected {
		t.Fatalf("digest=%s want=%s err=%v", digest, expected, err)
	}
	configuration = strings.Replace(configuration, "mode: enforce", "mode: advise", 1)
	if err = os.WriteFile(configPath, []byte(configuration), 0o600); err != nil {
		t.Fatal(err)
	}
	if code := evaluate([]string{"--config", configPath}); code != 0 {
		t.Fatalf("advise exit=%d want=0", code)
	}
}

func TestEvaluateToolResultControlsEnforcement(t *testing.T) {
	for _, test := range []struct {
		status string
		code   int
	}{{"passed", 0}, {"failed", 2}, {"pending", 3}} {
		t.Run(test.status, func(t *testing.T) {
			directory := t.TempDir()
			resultPath := filepath.Join(directory, "tool.json")
			if err := os.WriteFile(resultPath, []byte(`{"tool":"github-actions","status":"`+test.status+`"}`), 0o600); err != nil {
				t.Fatal(err)
			}
			configPath := filepath.Join(directory, ".epistemic.yaml")
			configuration := "api_version: epistemic.dev/v1alpha1\nmode: enforce\ndecision:\n  id: tool-" + test.status + "\n  recommendation: safe\n  action_type: source_change\n  subject_type: repository\n  subject_id: fixture\n  approved: true\nprovider:\n  type: local\nsources:\n  - type: tool\n    path: " + filepath.ToSlash(resultPath) + "\n"
			if err := os.WriteFile(configPath, []byte(configuration), 0o600); err != nil {
				t.Fatal(err)
			}
			if code := evaluate([]string{"--config", configPath}); code != test.code {
				t.Fatalf("enforce exit=%d want=%d", code, test.code)
			}
		})
	}
}
