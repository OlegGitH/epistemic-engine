package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/pipeline"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/service"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/store"
)

type Handler struct {
	service  *service.Service
	protocol *protocolState
	storage  string
	durable  bool
}

var requestCounter atomic.Uint64

type Option func(*Handler)

func WithStorage(storage string, durable bool) Option {
	return func(handler *Handler) {
		handler.storage = storage
		handler.durable = durable
	}
}

func New(s *service.Service, options ...Option) http.Handler {
	handler := &Handler{service: s, protocol: newProtocolState(), storage: "memory"}
	for _, option := range options {
		option(handler)
	}
	return handler
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	correlationID := r.Header.Get("X-Correlation-ID")
	if correlationID == "" {
		correlationID = fmt.Sprintf("req_%x_%x", started.UnixNano(), requestCounter.Add(1))
	}
	w.Header().Set("X-Correlation-ID", correlationID)
	defer func() {
		log.Printf(`{"level":"info","message":"http_request","correlation_id":%q,"method":%q,"path":%q,"duration_ms":%d}`, correlationID, r.Method, r.URL.Path, time.Since(started).Milliseconds())
	}()
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Correlation-ID, Epistemic-Context")
	w.Header().Set("Access-Control-Expose-Headers", "X-Correlation-ID, Epistemic-Context")
	w.Header().Set("Access-Control-Allow-Methods", "DELETE,GET,POST,OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if serveDocumentation(w, r) {
		return
	}
	if (r.URL.Path == "/health" || r.URL.Path == "/healthz") && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "storage": h.storage, "durable": h.durable})
		return
	}
	if r.URL.Path == "/.well-known/epistemic" && r.Method == http.MethodGet {
		h.protocolDiscovery(w)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "v1" {
		notFound(w)
		return
	}

	switch {
	case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "accounts":
		h.createAccount(w, r)
	case r.Method == http.MethodGet && len(parts) == 4 && parts[1] == "accounts" && parts[3] == "dashboard":
		h.accountDashboard(w, r, parts[2])
	case r.Method == http.MethodPost && len(parts) == 4 && parts[1] == "accounts" && parts[3] == "projects":
		h.createProject(w, r, parts[2])
	case r.Method == http.MethodPost && len(parts) == 4 && parts[1] == "projects" && parts[3] == "ai-systems":
		h.createAISystem(w, r, parts[2])
	case r.Method == http.MethodPost && len(parts) == 4 && parts[1] == "projects" && parts[3] == "connections":
		h.createProjectConnection(w, r, parts[2])
	case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "ingest":
		h.ingestProject(w, r)
	case r.Method == http.MethodDelete && len(parts) == 3 && parts[1] == "connections":
		h.revokeProjectConnection(w, r, parts[2])
	case r.Method == http.MethodGet && len(parts) == 2 && parts[1] == "tools":
		h.tools(w)
	case r.Method == http.MethodPost && len(parts) == 4 && parts[1] == "tools" && parts[2] == "github-actions" && parts[3] == "pipelines":
		h.generateGitHubPipeline(w, r)
	case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "events":
		h.protocolEmit(w, r)
	case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "events:batch":
		h.protocolBatch(w, r)
	case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "decisions:evaluate":
		h.protocolEvaluate(w, r)
	case r.Method == http.MethodGet && len(parts) == 2 && parts[1] == "stream":
		h.protocolStream(w, r)
	case r.Method == http.MethodGet && len(parts) == 3 && parts[1] == "decisions":
		h.protocolDecision(w, parts[2])
	case r.Method == http.MethodGet && len(parts) == 4 && parts[1] == "decisions" && parts[3] == "events":
		h.protocolEvents(w, parts[2])
	case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "runs":
		h.createRun(w, r)
	case len(parts) == 4 && parts[1] == "runs" && parts[3] == "events" && r.Method == http.MethodPost:
		h.addEvent(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "runs" && parts[3] == "analyze" && r.Method == http.MethodPost:
		h.analyze(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "runs" && parts[3] == "graph" && r.Method == http.MethodGet:
		h.graph(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "runs" && parts[3] == "stream" && r.Method == http.MethodGet:
		h.stream(w, r, parts[2])
	case len(parts) == 5 && parts[1] == "runs" && parts[3] == "events" && parts[4] == "stream" && r.Method == http.MethodGet:
		h.stream(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "decisions" && (parts[3] == "plan" || parts[3] == "verification-plan") && r.Method == http.MethodPost:
		h.plan(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "verifications" && parts[3] == "execute" && r.Method == http.MethodPost:
		h.execute(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "decisions" && parts[3] == "evaluate" && r.Method == http.MethodPost:
		h.evaluate(w, r, parts[2])
	case len(parts) == 4 && parts[1] == "decisions" && parts[3] == "certificate" && r.Method == http.MethodGet:
		if h.protocol.hasDecision(parts[2]) {
			h.protocolCertificate(w, parts[2])
		} else {
			h.certificate(w, r, parts[2])
		}
	case len(parts) == 5 && parts[1] == "decisions" && parts[3] == "certificate" && parts[4] == "report" && r.Method == http.MethodGet:
		h.certificateReport(w, r, parts[2])
	default:
		notFound(w)
	}
}

func (h *Handler) createAccount(w http.ResponseWriter, r *http.Request) {
	var input service.CreateAccountInput
	if !decode(w, r, &input) {
		return
	}
	value, err := h.service.CreateAccount(r.Context(), input)
	respond(w, value, err, http.StatusCreated)
}

func (h *Handler) accountDashboard(w http.ResponseWriter, r *http.Request, accountID string) {
	value, err := h.service.AccountDashboard(r.Context(), accountID)
	respond(w, value, err, http.StatusOK)
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request, accountID string) {
	var input service.CreateProjectInput
	if !decode(w, r, &input) {
		return
	}
	value, err := h.service.CreateProject(r.Context(), accountID, input)
	respond(w, value, err, http.StatusCreated)
}

func (h *Handler) createAISystem(w http.ResponseWriter, r *http.Request, projectID string) {
	var input service.CreateAISystemInput
	if !decode(w, r, &input) {
		return
	}
	value, err := h.service.CreateAISystem(r.Context(), projectID, input)
	respond(w, value, err, http.StatusCreated)
}

func (h *Handler) createProjectConnection(w http.ResponseWriter, r *http.Request, projectID string) {
	var input service.CreateConnectionInput
	if !decode(w, r, &input) {
		return
	}
	value, err := h.service.CreateProjectConnection(r.Context(), projectID, input)
	respond(w, value, err, http.StatusCreated)
}

func (h *Handler) ingestProject(w http.ResponseWriter, r *http.Request) {
	var input service.IngestInput
	if !decode(w, r, &input) {
		return
	}
	token := strings.TrimSpace(strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
	value, err := h.service.IngestProject(r.Context(), token, input)
	respond(w, value, err, http.StatusAccepted)
}

func (h *Handler) revokeProjectConnection(w http.ResponseWriter, r *http.Request, connectionID string) {
	value, err := h.service.RevokeProjectConnection(r.Context(), connectionID)
	respond(w, value, err, http.StatusOK)
}

func (h *Handler) tools(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, map[string]any{"tools": pipeline.Catalog()})
}

func (h *Handler) generateGitHubPipeline(w http.ResponseWriter, r *http.Request) {
	var input pipeline.GenerateInput
	if !decode(w, r, &input) {
		return
	}
	output, err := pipeline.GenerateGitHub(input)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusCreated, output)
}

func (h *Handler) createRun(w http.ResponseWriter, r *http.Request) {
	var in service.CreateRunInput
	if !decode(w, r, &in) {
		return
	}
	v, err := h.service.CreateRun(r.Context(), in)
	respond(w, v, err, http.StatusCreated)
}
func (h *Handler) addEvent(w http.ResponseWriter, r *http.Request, id string) {
	var in service.AddEventInput
	if !decode(w, r, &in) {
		return
	}
	if in.CorrelationID == "" {
		in.CorrelationID = w.Header().Get("X-Correlation-ID")
	}
	v, err := h.service.AddEvent(r.Context(), id, in)
	respond(w, v, err, http.StatusAccepted)
}
func (h *Handler) analyze(w http.ResponseWriter, r *http.Request, id string) {
	v, err := h.service.Analyze(r.Context(), id)
	respond(w, v, err, http.StatusOK)
}
func (h *Handler) graph(w http.ResponseWriter, r *http.Request, id string) {
	v, err := h.service.Graph(r.Context(), id)
	respond(w, v, err, http.StatusOK)
}
func (h *Handler) plan(w http.ResponseWriter, r *http.Request, id string) {
	v, err := h.service.Plan(r.Context(), id)
	respond(w, map[string]any{"verifications": v}, err, http.StatusCreated)
}
func (h *Handler) execute(w http.ResponseWriter, r *http.Request, id string) {
	var in service.ExecuteVerificationInput
	if !decode(w, r, &in) {
		return
	}
	v, err := h.service.ExecuteVerification(r.Context(), id, in)
	respond(w, v, err, http.StatusOK)
}
func (h *Handler) evaluate(w http.ResponseWriter, r *http.Request, id string) {
	var in struct {
		HumanApproved bool `json:"human_approved"`
	}
	if !decode(w, r, &in) {
		return
	}
	v, err := h.service.Evaluate(r.Context(), id, in.HumanApproved)
	respond(w, v, err, http.StatusOK)
}
func (h *Handler) certificate(w http.ResponseWriter, r *http.Request, id string) {
	v, err := h.service.Certificate(r.Context(), id)
	respond(w, v, err, http.StatusOK)
}
func (h *Handler) certificateReport(w http.ResponseWriter, r *http.Request, id string) {
	v, err := h.service.CertificateReport(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if r.URL.Query().Get("format") == "markdown" {
		w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
		w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="decision-report-%s.md"`, id))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(v.Markdown))
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func (h *Handler) stream(w http.ResponseWriter, r *http.Request, id string) {
	g, err := h.service.Graph(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming is unsupported"})
		return
	}
	payload, _ := json.Marshal(g)
	fmt.Fprintf(w, "event: graph.snapshot\ndata: %s\n\n", payload)
	flusher.Flush()
	events, unsubscribe := h.service.Subscribe(id)
	defer unsubscribe()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-events:
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, event.Data)
			flusher.Flush()
		case <-heartbeat.C:
			fmt.Fprint(w, ": heartbeat\n\n")
			flusher.Flush()
		}
	}
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON: " + err.Error()})
		return false
	}
	return true
}
func respond(w http.ResponseWriter, v any, err error, status int) {
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, status, v)
}
func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, store.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, service.ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, service.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, store.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, service.ErrUnauthorized):
		status = http.StatusUnauthorized
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func notFound(w http.ResponseWriter) {
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "route not found"})
}
