package main

import (
	"encoding/json"
	"log"
	"net/http"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
	"github.com/OlegGitH/epistemic-engine/sdk/go/providers/local"
)

func main() {
	provider := local.New()
	http.HandleFunc("/.well-known/epistemic", func(w http.ResponseWriter, _ *http.Request) {
		respond(w, epistemic.Capabilities{ProtocolVersions: []string{epistemic.Version}, Transports: []string{"http-json"}, EventTypes: epistemic.EventTypes, DecisionStatuses: []string{"allow", "block", "indeterminate"}, Features: []string{"single-event", "synchronous-evaluation"}, Limits: epistemic.Limits{MaxEventBytes: 2 << 20, MaxBatchSize: 1}})
	})
	http.HandleFunc("/v1/events", func(w http.ResponseWriter, r *http.Request) {
		var event epistemic.Event
		if json.NewDecoder(r.Body).Decode(&event) != nil || epistemic.ValidateEvent(event) != nil {
			http.Error(w, "invalid event", http.StatusBadRequest)
			return
		}
		if err := provider.Emit(r.Context(), event); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	})
	http.HandleFunc("/v1/decisions:evaluate", func(w http.ResponseWriter, r *http.Request) {
		var request epistemic.DecisionRequest
		if json.NewDecoder(r.Body).Decode(&request) != nil {
			http.Error(w, "invalid decision", http.StatusBadRequest)
			return
		}
		result, err := provider.Evaluate(r.Context(), request)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		respond(w, result)
	})
	log.Println("alternate compatible server listening on :8091")
	log.Fatal(http.ListenAndServe(":8091", nil))
}
func respond(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
