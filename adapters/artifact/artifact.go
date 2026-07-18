package artifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/OlegGitH/epistemic-engine/adapters/junit"
	"github.com/OlegGitH/epistemic-engine/adapters/sarif"
	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

type Source struct {
	Type string `yaml:"type"`
	Path string `yaml:"path"`
	Name string `yaml:"name"`
}

type ToolResult struct {
	Tool   string `json:"tool"`
	Status string `json:"status"`
}

func Event(source Source, context epistemic.Context, sequence int64) (epistemic.Event, error) {
	data, err := os.ReadFile(source.Path)
	if err != nil {
		return epistemic.Event{}, err
	}
	digest := sha256.Sum256(data)
	hash := hex.EncodeToString(digest[:])
	payload := map[string]any{"evidence_type": source.Type, "path": filepath.ToSlash(source.Path), "sha256": hash, "bytes": len(data)}
	eventType, subjectType := "evidence.discovered", "evidence"
	switch source.Type {
	case "tool":
		var result ToolResult
		if err := json.Unmarshal(data, &result); err != nil {
			return epistemic.Event{}, fmt.Errorf("decode tool result: %w", err)
		}
		if result.Tool == "" {
			return epistemic.Event{}, fmt.Errorf("tool result requires tool")
		}
		payload["result"] = json.RawMessage(data)
		payload["kind"] = "tool"
		switch result.Status {
		case "passed":
			eventType, subjectType = "verification.completed", "verification"
		case "failed", "error":
			eventType, subjectType = "verification.failed", "verification"
		case "indeterminate", "pending":
			eventType, subjectType = "verification.requested", "verification"
		default:
			return epistemic.Event{}, fmt.Errorf("unsupported tool status %q", result.Status)
		}
	case "junit":
		result, parseErr := junit.Parse(source.Path)
		if parseErr != nil {
			return epistemic.Event{}, parseErr
		}
		payload["result"] = result
		payload["kind"] = "test"
		if result.Failures+result.Errors > 0 {
			eventType, subjectType = "verification.failed", "verification"
		} else {
			eventType, subjectType = "verification.completed", "verification"
		}
	case "sarif":
		result, parseErr := sarif.Parse(source.Path)
		if parseErr != nil {
			return epistemic.Event{}, parseErr
		}
		payload["result"] = result
		payload["kind"] = "sarif"
		if result.Levels["error"] > 0 {
			eventType, subjectType = "contradiction.detected", "contradiction"
		}
	case "json":
		var parsed any
		payload["valid_json"] = json.Unmarshal(data, &parsed) == nil
	case "diff", "build", "migration", "log", "trace":
		payload["kind"] = source.Type
		payload["summary"] = fmt.Sprintf("%d-byte content-addressed %s artifact", len(data), source.Type)
	default:
		payload["kind"] = "custom"
	}
	encoded, _ := json.Marshal(payload)
	name := source.Name
	if name == "" {
		name = filepath.Base(source.Path)
	}
	return epistemic.Event{SpecVersion: epistemic.Version, ID: "evt-" + hash[:24], Type: eventType, Source: epistemic.Source{Name: "epistemic-cli", Version: epistemic.Version}, Subject: epistemic.Subject{Type: subjectType, ID: name}, Time: time.Now().UTC(), Context: context, Ordering: epistemic.Ordering{Sequence: sequence, Partition: context.RunID}, IdempotencyKey: hash, Data: encoded}, nil
}
