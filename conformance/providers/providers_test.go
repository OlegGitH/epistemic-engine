package providers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
	fileprovider "github.com/OlegGitH/epistemic-engine/sdk/go/providers/file"
	"github.com/OlegGitH/epistemic-engine/sdk/go/providers/local"
	"github.com/OlegGitH/epistemic-engine/sdk/go/providers/noop"
	"github.com/OlegGitH/epistemic-engine/sdk/go/providers/remote"
)

func TestP0Providers(t *testing.T) {
	t.Run("local", func(t *testing.T) { Run(t, func(*testing.T) epistemic.Provider { return local.New() }) })
	t.Run("noop", func(t *testing.T) { Run(t, func(*testing.T) epistemic.Provider { value := noop.New(); return value }) })
	t.Run("file", func(t *testing.T) {
		Run(t, func(t *testing.T) epistemic.Provider {
			return fileprovider.New(filepath.Join(t.TempDir(), "events.jsonl"))
		})
	})
	t.Run("remote", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			if r.URL.Path == "/v1/events" {
				w.WriteHeader(http.StatusAccepted)
				_, _ = w.Write([]byte(`{"accepted":true}`))
				return
			}
			if r.URL.Path == "/v1/decisions:evaluate" {
				var request epistemic.DecisionRequest
				_ = json.NewDecoder(r.Body).Decode(&request)
				_ = json.NewEncoder(w).Encode(epistemic.DecisionResult{SpecVersion: epistemic.Version, DecisionID: request.DecisionID, Status: "indeterminate", Reasons: []epistemic.Reason{}, Conditions: []string{}, EvaluatedAt: time.Now().UTC()})
				return
			}
			http.NotFound(w, r)
		}))
		defer server.Close()
		Run(t, func(*testing.T) epistemic.Provider { return remote.New(server.URL) })
	})
}
