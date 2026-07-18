package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"

	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
	"github.com/OlegGitH/epistemic-engine/sdk/go/providers/composite"
	fileprovider "github.com/OlegGitH/epistemic-engine/sdk/go/providers/file"
	"github.com/OlegGitH/epistemic-engine/sdk/go/providers/remote"
)

type relay struct {
	provider epistemic.Provider
	redact   map[string]bool
	maxBatch int
}

func main() {
	configPath := flag.String("config", "", "optional relay pipeline YAML")
	listen := flag.String("listen", ":8090", "HTTP listen address")
	archive := flag.String("archive", ".epistemic/relay.jsonl", "immutable JSONL archive")
	exporter := flag.String("export", "", "optional compatible HTTP exporter")
	watch := flag.String("watch", "", "optional artifact directory to watch")
	redact := flag.String("redact", "token,password,secret,authorization,api_key", "comma-separated JSON keys to redact")
	maxBatch := 100
	flag.Parse()
	if *configPath != "" {
		configuration, err := loadRelayConfig(*configPath)
		if err != nil {
			log.Fatal(err)
		}
		if configuration.Receivers.HTTP.Listen != "" {
			*listen = configuration.Receivers.HTTP.Listen
		}
		if configuration.Receivers.File.Watch != "" {
			*watch = configuration.Receivers.File.Watch
		}
		if configuration.Processors.Redact.Keys != "" {
			*redact = configuration.Processors.Redact.Keys
		}
		if configuration.Processors.Batch.MaxSize > 0 {
			maxBatch = configuration.Processors.Batch.MaxSize
		}
		if configuration.Exporters.Archive.Path != "" {
			*archive = configuration.Exporters.Archive.Path
		}
		if configuration.Exporters.Engine.Endpoint != "" {
			*exporter = configuration.Exporters.Engine.Endpoint
		}
	}
	providers := []epistemic.Provider{fileprovider.New(*archive)}
	if *exporter != "" {
		providers = append(providers, &retryProvider{provider: remote.New(*exporter), attempts: 3})
	}
	r := &relay{provider: composite.New(providers...), redact: map[string]bool{}, maxBatch: maxBatch}
	for _, key := range strings.Split(*redact, ",") {
		r.redact[strings.ToLower(strings.TrimSpace(key))] = true
	}
	server := &http.Server{Addr: *listen, Handler: r, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if *watch != "" {
		go r.watch(ctx, *watch)
	}
	go func() {
		log.Printf("epistemic relay listening on %s", *listen)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
	_ = r.provider.Flush(shutdownCtx)
	_ = r.provider.Shutdown(shutdownCtx)
}

type relayConfig struct {
	APIVersion string `yaml:"api_version"`
	Receivers  struct {
		HTTP struct {
			Listen string `yaml:"listen"`
		} `yaml:"http"`
		File struct {
			Watch string `yaml:"watch"`
		} `yaml:"file"`
	} `yaml:"receivers"`
	Processors struct {
		Redact struct {
			Keys string `yaml:"keys"`
		} `yaml:"redact"`
		Batch struct {
			MaxSize int `yaml:"max_size"`
		} `yaml:"batch"`
	} `yaml:"processors"`
	Exporters struct {
		Archive struct {
			Path string `yaml:"path"`
		} `yaml:"archive"`
		Engine struct {
			Endpoint string `yaml:"endpoint"`
		} `yaml:"engine"`
	} `yaml:"exporters"`
}

func loadRelayConfig(path string) (relayConfig, error) {
	var configuration relayConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return configuration, err
	}
	if err = yaml.Unmarshal(data, &configuration); err != nil {
		return configuration, err
	}
	if configuration.APIVersion != "epistemic.dev/relay/v1alpha1" {
		return configuration, fmt.Errorf("unsupported relay api_version %q", configuration.APIVersion)
	}
	return configuration, nil
}

func (r *relay) ServeHTTP(w http.ResponseWriter, request *http.Request) {
	if request.Method == http.MethodGet && request.URL.Path == "/.well-known/epistemic" {
		write(w, http.StatusOK, epistemic.Capabilities{ProtocolVersions: []string{epistemic.Version}, Transports: []string{"http-json"}, EventTypes: epistemic.EventTypes, DecisionStatuses: []string{"allow", "block", "indeterminate", "error"}, Features: []string{"relay", "redaction", "archive", "retry", "batch", "file-watch"}, Limits: epistemic.Limits{MaxEventBytes: 2 << 20, MaxBatchSize: r.maxBatch}})
		return
	}
	if request.Method == http.MethodPost && request.URL.Path == "/v1/events" {
		var event epistemic.Event
		if !decode(w, request, &event) {
			return
		}
		if err := r.process(request.Context(), &event); err != nil {
			protocolError(w, err)
			return
		}
		write(w, http.StatusAccepted, map[string]any{"accepted": true, "id": event.ID})
		return
	}
	if request.Method == http.MethodPost && request.URL.Path == "/v1/events:batch" {
		var body struct {
			Events []epistemic.Event `json:"events"`
		}
		if !decode(w, request, &body) {
			return
		}
		if len(body.Events) > r.maxBatch {
			write(w, http.StatusRequestEntityTooLarge, epistemic.Error{SpecVersion: epistemic.Version, Code: "limit_exceeded", Message: "batch too large"})
			return
		}
		ids := []string{}
		for index := range body.Events {
			if err := r.process(request.Context(), &body.Events[index]); err != nil {
				protocolError(w, err)
				return
			}
			ids = append(ids, body.Events[index].ID)
		}
		write(w, http.StatusAccepted, map[string]any{"accepted": ids, "duplicate": []string{}, "errors": map[string]string{}})
		return
	}
	if request.Method == http.MethodPost && request.URL.Path == "/v1/decisions:evaluate" {
		var decision epistemic.DecisionRequest
		if !decode(w, request, &decision) {
			return
		}
		for index := range decision.Events {
			r.redactEvent(&decision.Events[index])
		}
		result, err := r.provider.Evaluate(request.Context(), decision)
		if err != nil {
			protocolError(w, err)
			return
		}
		write(w, http.StatusOK, result)
		return
	}
	write(w, http.StatusNotFound, epistemic.Error{SpecVersion: epistemic.Version, Code: "not_found", Message: "route not found"})
}

func (r *relay) process(ctx context.Context, event *epistemic.Event) error {
	if err := epistemic.ValidateEvent(*event); err != nil {
		return err
	}
	r.redactEvent(event)
	return r.provider.Emit(ctx, *event)
}
func (r *relay) redactEvent(event *epistemic.Event) {
	var value any
	if json.Unmarshal(event.Data, &value) != nil {
		return
	}
	event.Data = mustJSON(r.redactValue(value))
}
func (r *relay) redactValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		for key, item := range typed {
			if r.redact[strings.ToLower(key)] {
				typed[key] = "[REDACTED]"
			} else {
				typed[key] = r.redactValue(item)
			}
		}
	case []any:
		for index := range typed {
			typed[index] = r.redactValue(typed[index])
		}
	}
	return value
}

func (r *relay) watch(ctx context.Context, root string) {
	seen := map[string]string{}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		r.scan(ctx, root, seen)
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
func (r *relay) scan(ctx context.Context, root string, seen map[string]string) {
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		sum := sha256.Sum256(data)
		digest := hex.EncodeToString(sum[:])
		if seen[path] == digest {
			return nil
		}
		seen[path] = digest
		payload := mustJSON(map[string]any{"evidence_type": "custom", "path": filepath.ToSlash(path), "sha256": digest, "bytes": len(data)})
		event := epistemic.Event{SpecVersion: epistemic.Version, ID: "file-" + digest[:24], Type: "evidence.discovered", Source: epistemic.Source{Name: "epistemic-relay-file-receiver", Version: epistemic.Version}, Subject: epistemic.Subject{Type: "artifact", ID: filepath.Base(path)}, Time: time.Now().UTC(), IdempotencyKey: digest, Data: payload}
		if emitErr := r.provider.Emit(ctx, event); emitErr != nil {
			log.Printf("watch emit: %v", emitErr)
		}
		return nil
	})
}

type retryProvider struct {
	provider epistemic.Provider
	attempts int
}

func (p *retryProvider) Emit(ctx context.Context, event epistemic.Event) error {
	return retry(ctx, p.attempts, func() error { return p.provider.Emit(ctx, event) })
}
func (p *retryProvider) Evaluate(ctx context.Context, request epistemic.DecisionRequest) (result epistemic.DecisionResult, err error) {
	err = retry(ctx, p.attempts, func() error { result, err = p.provider.Evaluate(ctx, request); return err })
	return
}
func (p *retryProvider) Flush(ctx context.Context) error    { return p.provider.Flush(ctx) }
func (p *retryProvider) Shutdown(ctx context.Context) error { return p.provider.Shutdown(ctx) }
func retry(ctx context.Context, attempts int, operation func() error) error {
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		if err = operation(); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(attempt+1) * 100 * time.Millisecond):
		}
	}
	return err
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		protocolError(w, err)
		return false
	}
	return true
}
func protocolError(w http.ResponseWriter, err error) {
	write(w, http.StatusBadRequest, epistemic.Error{SpecVersion: epistemic.Version, Code: "invalid_message", Message: err.Error(), Retryable: false})
}
func write(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
func mustJSON(value any) json.RawMessage { data, _ := json.Marshal(value); return data }
