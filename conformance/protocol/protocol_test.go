package protocol_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
	"gopkg.in/yaml.v3"
)

func TestCanonicalProtocolFixtures(t *testing.T) {
	fixtures := filepath.Join("..", "fixtures")
	validData, err := os.ReadFile(filepath.Join(fixtures, "valid-event.json"))
	if err != nil {
		t.Fatal(err)
	}
	var valid epistemic.Event
	if err = json.Unmarshal(validData, &valid); err != nil {
		t.Fatal(err)
	}
	if err = epistemic.ValidateEvent(valid); err != nil {
		t.Fatalf("valid fixture rejected: %v", err)
	}
	invalidData, err := os.ReadFile(filepath.Join(fixtures, "invalid-event.json"))
	if err != nil {
		t.Fatal(err)
	}
	var invalid epistemic.Event
	if err = json.Unmarshal(invalidData, &invalid); err != nil {
		t.Fatal(err)
	}
	if err = epistemic.ValidateEvent(invalid); err == nil {
		t.Fatal("invalid fixture was accepted")
	}
	var canonical struct {
		Value  any    `json:"value"`
		SHA256 string `json:"sha256"`
	}
	data, err := os.ReadFile(filepath.Join(fixtures, "canonical.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &canonical); err != nil {
		t.Fatal(err)
	}
	digest, err := epistemic.Hash(canonical.Value)
	if err != nil {
		t.Fatal(err)
	}
	if digest != canonical.SHA256 {
		t.Fatalf("canonical digest=%s want=%s", digest, canonical.SHA256)
	}
}

func TestSchemasOpenAPIAndReleaseVersionsAlign(t *testing.T) {
	root := filepath.Join("..", "..")
	schemas, err := filepath.Glob(filepath.Join(root, "specification", "schemas", "v0.1", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(schemas) < 14 {
		t.Fatalf("expected base and eight family schemas, got %d", len(schemas))
	}
	for _, path := range schemas {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			t.Fatal(readErr)
		}
		var document any
		if json.Unmarshal(data, &document) != nil {
			t.Errorf("invalid schema JSON: %s", path)
		}
	}
	openAPI, err := os.ReadFile(filepath.Join(root, "specification", "openapi-v0.1.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var contract map[string]any
	if err = yaml.Unmarshal(openAPI, &contract); err != nil {
		t.Fatal(err)
	}
	paths, ok := contract["paths"].(map[string]any)
	if !ok || len(paths) != 8 {
		t.Fatalf("expected 8 HTTP endpoints, got %d", len(paths))
	}
	releaseData, err := os.ReadFile(filepath.Join(root, "release", "v0.1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var release map[string]string
	if err = json.Unmarshal(releaseData, &release); err != nil {
		t.Fatal(err)
	}
	if release["protocol"] != epistemic.Version || release["schemas"] != epistemic.Version {
		t.Fatalf("release versions are not aligned: %+v", release)
	}
}

func TestAllEightEventFamiliesAreRepresented(t *testing.T) {
	families := map[string]bool{}
	for _, eventType := range epistemic.EventTypes {
		for index, value := range eventType {
			if value == '.' {
				families[eventType[:index]] = true
				break
			}
		}
	}
	for _, expected := range []string{"decision", "claim", "evidence", "assumption", "unknown", "contradiction", "verification", "proof"} {
		if !families[expected] {
			t.Errorf("missing event family %s", expected)
		}
	}
}

func TestPortableExampleEventsValidate(t *testing.T) {
	paths := []string{filepath.Join("..", "..", "examples", "agent-tool-execution", "events.json"), filepath.Join("..", "..", "examples", "research-evidence", "events.json")}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var events []epistemic.Event
		if err = json.Unmarshal(data, &events); err != nil {
			t.Fatal(err)
		}
		for index, event := range events {
			if err = epistemic.ValidateEvent(event); err != nil {
				t.Errorf("%s event %d: %v", path, index, err)
			}
		}
	}
}
