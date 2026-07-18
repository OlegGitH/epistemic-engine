package artifact

import (
	"os"
	"path/filepath"
	"testing"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

func TestToolResultMapsStatusToVerificationEvent(t *testing.T) {
	directory := t.TempDir()
	for _, test := range []struct {
		status string
		want   string
	}{{"passed", "verification.completed"}, {"failed", "verification.failed"}, {"pending", "verification.requested"}} {
		path := filepath.Join(directory, test.status+".json")
		if err := os.WriteFile(path, []byte(`{"tool":"go-test","status":"`+test.status+`"}`), 0o600); err != nil {
			t.Fatal(err)
		}
		event, err := Event(Source{Type: "tool", Path: path}, epistemic.Context{DecisionID: "decision"}, 1)
		if err != nil {
			t.Fatal(err)
		}
		if event.Type != test.want {
			t.Fatalf("status %s mapped to %s, want %s", test.status, event.Type, test.want)
		}
	}
}

func TestToolResultRejectsUnknownStatus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "result.json")
	if err := os.WriteFile(path, []byte(`{"tool":"example-check","status":"maybe"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Event(Source{Type: "tool", Path: path}, epistemic.Context{}, 1); err == nil {
		t.Fatal("expected unknown status to fail")
	}
}
