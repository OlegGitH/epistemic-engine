package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

type run struct {
	ID         string `json:"id"`
	DecisionID string `json:"decision_id"`
}

type graph struct {
	Claims        []json.RawMessage `json:"claims"`
	Evidence      []json.RawMessage `json:"evidence"`
	Verifications []json.RawMessage `json:"verifications"`
}

func main() {
	baseURL := flag.String("api", "http://localhost:8080", "control-plane base URL")
	scenario := flag.String("scenario", "unsafe", "unsafe, corrected, or pending")
	flag.Parse()

	events, ok := scenarios()[*scenario]
	if !ok {
		fatal(fmt.Errorf("unknown scenario %q", *scenario))
	}
	created := run{}
	post(*baseURL+"/v1/runs", map[string]any{
		"external_trace_id": fmt.Sprintf("seed-%s-%d", *scenario, time.Now().UnixNano()),
		"title":             "Orders deployment — " + *scenario,
		"source":            "seed-cli",
		"recommendation":    "The orders change is safe to deploy.",
		"action_type":       "software_deployment",
		"subject":           "demo/unsafe-orders-pr",
		"risk_level":        "high",
	}, &created)
	for index, event := range events {
		event["external_id"] = fmt.Sprintf("seed-%s-%02d", *scenario, index+1)
		event["sequence"] = index + 1
		post(fmt.Sprintf("%s/v1/runs/%s/events", *baseURL, created.ID), event, &map[string]any{})
	}
	analyzed := graph{}
	post(fmt.Sprintf("%s/v1/runs/%s/analyze", *baseURL, created.ID), nil, &analyzed)
	fmt.Printf("run_id=%s\ndecision_id=%s\nclaims=%d evidence=%d\n", created.ID, created.DecisionID, len(analyzed.Claims), len(analyzed.Evidence))
}

func scenarios() map[string][]map[string]any {
	passed := func(kind string, details map[string]any) map[string]any {
		return map[string]any{"type": kind, "source": "demo-ci", "payload": details}
	}
	return map[string][]map[string]any{
		"unsafe": {
			passed("build.completed", map[string]any{"status": "passed", "revision": "unsafe-orders"}),
			passed("test.completed", map[string]any{"status": "passed", "suite": "unit"}),
			passed("migration.test.completed", map[string]any{"status": "failed", "case": "legacy processing status"}),
			passed("code.diff.observed", map[string]any{"path": "orders/service.py", "patch": `logger.info("customer_email=alice@example.com")`}),
		},
		"pending": {
			passed("build.completed", map[string]any{"status": "passed", "revision": "corrected-orders"}),
			passed("test.completed", map[string]any{"status": "passed", "suite": "unit"}),
		},
		"corrected": {
			passed("build.completed", map[string]any{"status": "passed", "revision": "corrected-orders"}),
			passed("test.completed", map[string]any{"status": "passed", "suite": "unit"}),
			passed("compatibility.test.completed", map[string]any{"status": "passed", "case": "legacy processing status"}),
			passed("pii.test.completed", map[string]any{"status": "passed", "case": "email redaction"}),
			passed("rollback.check.completed", map[string]any{"status": "passed", "strategy": "staged rollback ready"}),
		},
	}
}

func post(url string, value any, target any) {
	var body io.Reader
	if value != nil {
		data, err := json.Marshal(value)
		if err != nil {
			fatal(err)
		}
		body = bytes.NewReader(data)
	}
	request, err := http.NewRequest(http.MethodPost, url, body)
	if err != nil {
		fatal(err)
	}
	if value != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		fatal(err)
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		fatal(fmt.Errorf("POST %s: %s: %s", url, response.Status, strings.TrimSpace(string(data))))
	}
	if err := json.Unmarshal(data, target); err != nil {
		fatal(fmt.Errorf("decode POST %s: %w", url, err))
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
