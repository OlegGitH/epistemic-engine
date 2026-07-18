package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/service"
	epistemic "github.com/OlegGitH/epistemic-engine/api/go"
)

const maxProtocolBatch = 100

type protocolState struct {
	mu               sync.RWMutex
	events           map[string]epistemic.Event
	idempotency      map[string]string
	eventsByDecision map[string][]string
	results          map[string]epistemic.DecisionResult
	certificates     map[string]epistemic.Certificate
	runs             map[string]string
	engineDecisions  map[string]string
	subscribers      map[int]chan epistemic.Event
	sequences        map[string]map[int64]string
	nextSubscriber   int
}

func newProtocolState() *protocolState {
	return &protocolState{events: map[string]epistemic.Event{}, idempotency: map[string]string{}, eventsByDecision: map[string][]string{}, results: map[string]epistemic.DecisionResult{}, certificates: map[string]epistemic.Certificate{}, runs: map[string]string{}, engineDecisions: map[string]string{}, subscribers: map[int]chan epistemic.Event{}, sequences: map[string]map[int64]string{}}
}

func (s *protocolState) hasDecision(id string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, result := s.results[id]
	return result
}

func (h *Handler) protocolDiscovery(w http.ResponseWriter) {
	writeJSON(w, http.StatusOK, epistemic.Capabilities{ProtocolVersions: []string{epistemic.Version}, Transports: []string{"http-json", "sse"}, EventTypes: epistemic.EventTypes, DecisionStatuses: []string{"allow", "block", "indeterminate", "error"}, Features: []string{"single-event", "batch", "synchronous-evaluation", "decision-query", "event-history", "certificate", "stream", "idempotency", "ordering", "context-propagation"}, Limits: epistemic.Limits{MaxEventBytes: 2 << 20, MaxBatchSize: maxProtocolBatch}})
}

func (h *Handler) protocolEmit(w http.ResponseWriter, r *http.Request) {
	var event epistemic.Event
	if !decodeProtocol(w, r, &event) {
		return
	}
	mergeContextHeader(&event.Context, r.Header.Get("Epistemic-Context"))
	setContextResponse(w, event.Context)
	created, err := h.acceptProtocolEvent(r, event)
	if err != nil {
		writeProtocolError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": created, "duplicate": !created, "id": event.ID})
}

func (h *Handler) protocolBatch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Events []epistemic.Event `json:"events"`
	}
	if !decodeProtocol(w, r, &body) {
		return
	}
	if len(body.Events) > maxProtocolBatch {
		writeProtocolErrorCode(w, http.StatusRequestEntityTooLarge, "limit_exceeded", "batch exceeds advertised maximum", false)
		return
	}
	accepted, duplicates := []string{}, []string{}
	errorsByID := map[string]string{}
	for index := range body.Events {
		mergeContextHeader(&body.Events[index].Context, r.Header.Get("Epistemic-Context"))
		created, err := h.acceptProtocolEvent(r, body.Events[index])
		if err != nil {
			errorsByID[body.Events[index].ID] = err.Error()
			continue
		}
		if created {
			accepted = append(accepted, body.Events[index].ID)
		} else {
			duplicates = append(duplicates, body.Events[index].ID)
		}
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"accepted": accepted, "duplicate": duplicates, "errors": errorsByID})
}

func (h *Handler) acceptProtocolEvent(r *http.Request, event epistemic.Event) (bool, error) {
	if err := epistemic.ValidateEvent(event); err != nil {
		return false, classifyProtocolValidation(err)
	}
	key := event.IdempotencyKey
	if key == "" {
		key = event.ID
	}
	h.protocol.mu.Lock()
	if existing, exists := h.protocol.events[event.ID]; exists {
		h.protocol.mu.Unlock()
		existingHash, _ := epistemic.Hash(existing)
		incomingHash, _ := epistemic.Hash(event)
		if existingHash != incomingHash {
			return false, protocolFailure{Status: http.StatusConflict, Code: "conflict", Message: fmt.Sprintf("event id %s was reused with different content", event.ID)}
		}
		return false, nil
	}
	if existingID, exists := h.protocol.idempotency[key]; exists {
		h.protocol.mu.Unlock()
		if existingID != event.ID {
			return false, protocolFailure{Status: http.StatusConflict, Code: "conflict", Message: fmt.Sprintf("idempotency key already belongs to event %s", existingID)}
		}
		return false, nil
	}
	if event.Context.ParentID != "" {
		parent, exists := h.protocol.events[event.Context.ParentID]
		if !exists {
			h.protocol.mu.Unlock()
			return false, fmt.Errorf("parent event %s is unknown", event.Context.ParentID)
		}
		if event.Ordering.Sequence > 0 && parent.Ordering.Sequence > 0 && parent.Ordering.Partition == event.Ordering.Partition && parent.Ordering.Sequence >= event.Ordering.Sequence {
			h.protocol.mu.Unlock()
			return false, fmt.Errorf("parent event must precede child sequence")
		}
	}
	partition := event.Ordering.Partition
	if partition == "" {
		partition = event.Context.RunID
		if partition == "" {
			partition = event.Context.DecisionID
		}
	}
	if event.Ordering.Sequence > 0 && partition != "" {
		if h.protocol.sequences[partition] == nil {
			h.protocol.sequences[partition] = map[int64]string{}
		}
		if existingID := h.protocol.sequences[partition][event.Ordering.Sequence]; existingID != "" && existingID != event.ID {
			h.protocol.mu.Unlock()
			return false, fmt.Errorf("ordering sequence %d in partition %s belongs to event %s", event.Ordering.Sequence, partition, existingID)
		}
		h.protocol.sequences[partition][event.Ordering.Sequence] = event.ID
	}
	h.protocol.events[event.ID] = event
	h.protocol.idempotency[key] = event.ID
	if event.Context.DecisionID != "" {
		h.protocol.eventsByDecision[event.Context.DecisionID] = append(h.protocol.eventsByDecision[event.Context.DecisionID], event.ID)
	}
	runID := h.protocol.runs[event.Context.DecisionID]
	subscribers := make([]chan epistemic.Event, 0, len(h.protocol.subscribers))
	for _, subscriber := range h.protocol.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	h.protocol.mu.Unlock()
	if runID != "" {
		_, err := h.service.AddEvent(r.Context(), runID, service.AddEventInput{ExternalID: event.ID, Sequence: event.Ordering.Sequence, Type: event.Type, Source: event.Source.Name, CorrelationID: event.Context.Correlation, OccurredAt: event.Time, Payload: event.Data})
		if err != nil {
			h.protocol.mu.Lock()
			delete(h.protocol.events, event.ID)
			delete(h.protocol.idempotency, key)
			if event.Ordering.Sequence > 0 && partition != "" {
				delete(h.protocol.sequences[partition], event.Ordering.Sequence)
			}
			ids := h.protocol.eventsByDecision[event.Context.DecisionID]
			for index, id := range ids {
				if id == event.ID {
					h.protocol.eventsByDecision[event.Context.DecisionID] = append(ids[:index], ids[index+1:]...)
					break
				}
			}
			h.protocol.mu.Unlock()
			return false, err
		}
	}
	for _, subscriber := range subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
	return true, nil
}

func (h *Handler) protocolEvaluate(w http.ResponseWriter, r *http.Request) {
	var request epistemic.DecisionRequest
	if !decodeProtocol(w, r, &request) {
		return
	}
	mergeContextHeader(&request.Context, r.Header.Get("Epistemic-Context"))
	setContextResponse(w, request.Context)
	if err := epistemic.ValidateDecisionRequest(request); err != nil {
		writeProtocolError(w, classifyProtocolValidation(err))
		return
	}
	decisionID := request.DecisionID
	if decisionID == "" {
		decisionID = fmt.Sprintf("decision-%x", time.Now().UnixNano())
	}
	request.DecisionID = decisionID
	request.Context.DecisionID = decisionID
	h.protocol.mu.RLock()
	existing, exists := h.protocol.results[decisionID]
	h.protocol.mu.RUnlock()
	if exists {
		writeJSON(w, http.StatusOK, existing)
		return
	}
	// The Engine store requires external trace IDs to be unique. A portable run may
	// legitimately contain several decisions, so scope the storage key by decision
	// while preserving the caller's original run ID in the protocol context.
	externalTraceID := decisionID
	if request.Context.RunID != "" {
		externalTraceID = request.Context.RunID + ":" + decisionID
	}
	run, err := h.service.CreateRun(r.Context(), service.CreateRunInput{ExternalTraceID: externalTraceID, Title: request.Action.Subject.Type + " " + request.Action.Subject.ID, Goal: "Evaluate a portable Epistemic Protocol decision request.", Source: "epistemic-protocol/v0.1", Recommendation: request.Recommendation, ActionType: request.Action.Type, Subject: request.Action.Subject.ID, RiskLevel: normalizedRisk(request.Action.RiskLevel)})
	if err != nil {
		writeProtocolErrorCode(w, http.StatusInternalServerError, "evaluation_failed", err.Error(), false)
		return
	}
	if request.Context.RunID == "" {
		request.Context.RunID = run.ID
	}
	setContextResponse(w, request.Context)
	h.protocol.mu.Lock()
	h.protocol.runs[decisionID] = run.ID
	h.protocol.engineDecisions[decisionID] = run.DecisionID
	bufferedIDs := append([]string{}, h.protocol.eventsByDecision[decisionID]...)
	h.protocol.mu.Unlock()
	for _, eventID := range bufferedIDs {
		h.protocol.mu.RLock()
		event := h.protocol.events[eventID]
		h.protocol.mu.RUnlock()
		if _, addErr := h.service.AddEvent(r.Context(), run.ID, service.AddEventInput{ExternalID: event.ID, Sequence: event.Ordering.Sequence, Type: event.Type, Source: event.Source.Name, CorrelationID: event.Context.Correlation, OccurredAt: event.Time, Payload: event.Data}); addErr != nil {
			writeProtocolErrorCode(w, http.StatusConflict, "conflict", addErr.Error(), false)
			return
		}
	}
	for index := range request.Events {
		request.Events[index].Context.DecisionID = decisionID
		if request.Events[index].Context.RunID == "" {
			request.Events[index].Context.RunID = run.ID
		}
		if _, err = h.acceptProtocolEvent(r, request.Events[index]); err != nil {
			writeProtocolError(w, err)
			return
		}
	}
	graph, err := h.service.Analyze(r.Context(), run.ID)
	if err != nil {
		writeProtocolErrorCode(w, http.StatusInternalServerError, "evaluation_failed", err.Error(), false)
		return
	}
	engineCertificate, err := h.service.Evaluate(r.Context(), graph.Decision.ID, request.Approval.Approved)
	if err != nil {
		writeProtocolErrorCode(w, http.StatusInternalServerError, "evaluation_failed", err.Error(), false)
		return
	}
	result := portableResult(decisionID, request.Context, engineCertificate, graph)
	certificate, err := portableCertificate(result, engineCertificate)
	if err != nil {
		writeProtocolErrorCode(w, http.StatusInternalServerError, "evaluation_failed", err.Error(), false)
		return
	}
	result.Certificate = &certificate
	h.protocol.mu.Lock()
	h.protocol.results[decisionID] = result
	h.protocol.certificates[decisionID] = certificate
	h.protocol.mu.Unlock()
	h.emitGeneratedProtocolEvent(r, decisionID, run.ID, "decision.evaluated", "decision", decisionID, result)
	if result.Status == "block" {
		h.emitGeneratedProtocolEvent(r, decisionID, run.ID, "decision.blocked", "decision", decisionID, result.Reasons)
	}
	if result.ActionAllowed {
		h.emitGeneratedProtocolEvent(r, decisionID, run.ID, "decision.approved", "decision", decisionID, request.Approval)
	}
	h.emitGeneratedProtocolEvent(r, decisionID, run.ID, "proof.issued", "proof", certificate.ID, certificate)
	writeJSON(w, http.StatusOK, result)
}

func portableResult(id string, context epistemic.Context, certificate domain.Certificate, graph domain.Graph) epistemic.DecisionResult {
	conditions := append([]string{}, certificate.Conditions...)
	if conditions == nil {
		conditions = []string{}
	}
	result := epistemic.DecisionResult{SpecVersion: epistemic.Version, DecisionID: id, Status: "block", ActionAllowed: certificate.ActionAllowed, Reasons: []epistemic.Reason{}, Conditions: conditions, EvaluatedAt: certificate.IssuedAt, Context: context}
	switch certificate.Verdict {
	case domain.VerdictVerified, domain.VerdictVerifiedWithConditions:
		result.Status = "allow"
	case domain.VerdictContradicted:
		result.Reasons = append(result.Reasons, epistemic.Reason{Code: "contradicted", Message: "A critical claim is contradicted."})
	case domain.VerdictRejected:
		result.Reasons = append(result.Reasons, epistemic.Reason{Code: "rejected", Message: "A critical claim was rejected."})
	default:
		result.Reasons = append(result.Reasons, epistemic.Reason{Code: "insufficient_evidence", Message: "Critical evidence or verification is incomplete."})
	}
	for _, claim := range graph.Claims {
		if claim.State == domain.ClaimContradicted {
			result.Reasons = append(result.Reasons, epistemic.Reason{Code: "contradicted", Message: claim.Statement, SubjectID: claim.ID})
		}
	}
	if result.Status == "allow" && !result.ActionAllowed {
		result.Conditions = append(result.Conditions, "Human approval is required before consequential action.")
	}
	return result
}

func portableCertificate(result epistemic.DecisionResult, internal domain.Certificate) (epistemic.Certificate, error) {
	unsignedResult := result
	unsignedResult.Certificate = nil
	resultJSON, err := json.Marshal(unsignedResult)
	if err != nil {
		return epistemic.Certificate{}, err
	}
	certificate := epistemic.Certificate{SpecVersion: epistemic.Version, ID: "proof-" + result.DecisionID, DecisionID: result.DecisionID, Result: resultJSON, ArtifactHashes: internal.ArtifactHashes, IssuedAt: internal.IssuedAt, Proof: epistemic.Proof{Algorithm: "SHA-256"}}
	digest, err := epistemic.Hash(certificate)
	if err != nil {
		return certificate, err
	}
	certificate.Proof.Digest = digest
	return certificate, nil
}

func (h *Handler) emitGeneratedProtocolEvent(r *http.Request, decisionID, runID, eventType, subjectType, subjectID string, data any) {
	payload, _ := json.Marshal(data)
	event := epistemic.Event{SpecVersion: epistemic.Version, ID: fmt.Sprintf("%s-%x", strings.ReplaceAll(eventType, ".", "-"), time.Now().UnixNano()), Type: eventType, Source: epistemic.Source{Name: "epistemic-engine", Version: "0.1"}, Subject: epistemic.Subject{Type: subjectType, ID: subjectID}, Time: time.Now().UTC(), Context: epistemic.Context{DecisionID: decisionID, RunID: runID}, Data: payload}
	_, _ = h.acceptProtocolEvent(r, event)
}

func (h *Handler) protocolDecision(w http.ResponseWriter, id string) {
	h.protocol.mu.RLock()
	result, exists := h.protocol.results[id]
	h.protocol.mu.RUnlock()
	if !exists {
		writeProtocolErrorCode(w, http.StatusNotFound, "not_found", "decision not found", false)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) protocolEvents(w http.ResponseWriter, id string) {
	h.protocol.mu.RLock()
	ids := append([]string{}, h.protocol.eventsByDecision[id]...)
	events := make([]epistemic.Event, 0, len(ids))
	for _, eventID := range ids {
		events = append(events, h.protocol.events[eventID])
	}
	h.protocol.mu.RUnlock()
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Ordering.Sequence == events[j].Ordering.Sequence {
			return events[i].Time.Before(events[j].Time)
		}
		if events[i].Ordering.Sequence == 0 {
			return false
		}
		if events[j].Ordering.Sequence == 0 {
			return true
		}
		return events[i].Ordering.Sequence < events[j].Ordering.Sequence
	})
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

func (h *Handler) protocolCertificate(w http.ResponseWriter, id string) {
	h.protocol.mu.RLock()
	certificate, exists := h.protocol.certificates[id]
	h.protocol.mu.RUnlock()
	if !exists {
		writeProtocolErrorCode(w, http.StatusNotFound, "not_found", "certificate not found", false)
		return
	}
	writeJSON(w, http.StatusOK, certificate)
}

func (h *Handler) protocolStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeProtocolErrorCode(w, http.StatusInternalServerError, "temporarily_unavailable", "streaming unsupported", true)
		return
	}
	h.protocol.mu.Lock()
	id := h.protocol.nextSubscriber
	h.protocol.nextSubscriber++
	channel := make(chan epistemic.Event, 32)
	h.protocol.subscribers[id] = channel
	h.protocol.mu.Unlock()
	defer func() {
		h.protocol.mu.Lock()
		delete(h.protocol.subscribers, id)
		close(channel)
		h.protocol.mu.Unlock()
	}()
	fmt.Fprint(w, ": epistemic protocol v0.1\n\n")
	flusher.Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case event := <-channel:
			payload, _ := json.Marshal(event)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, payload)
			flusher.Flush()
		}
	}
}

func normalizedRisk(value string) string {
	switch value {
	case "low", "medium", "high", "critical":
		return value
	default:
		return "high"
	}
}

func mergeContextHeader(context *epistemic.Context, header string) {
	for _, item := range strings.Split(header, ";") {
		pair := strings.SplitN(strings.TrimSpace(item), "=", 2)
		if len(pair) != 2 {
			continue
		}
		switch pair[0] {
		case "decision":
			if context.DecisionID == "" {
				context.DecisionID = pair[1]
			}
		case "run":
			if context.RunID == "" {
				context.RunID = pair[1]
			}
		case "correlation":
			if context.Correlation == "" {
				context.Correlation = pair[1]
			}
		case "parent":
			if context.ParentID == "" {
				context.ParentID = pair[1]
			}
		}
	}
}

func setContextResponse(w http.ResponseWriter, context epistemic.Context) {
	parts := []string{}
	for _, pair := range [][2]string{{"decision", context.DecisionID}, {"run", context.RunID}, {"correlation", context.Correlation}, {"parent", context.ParentID}} {
		if pair[1] != "" {
			parts = append(parts, pair[0]+"="+pair[1])
		}
	}
	if len(parts) > 0 {
		w.Header().Set("Epistemic-Context", strings.Join(parts, ";"))
	}
}

func decodeProtocol(w http.ResponseWriter, r *http.Request, target any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(target); err != nil {
		writeProtocolErrorCode(w, http.StatusBadRequest, "invalid_message", "invalid JSON: "+err.Error(), false)
		return false
	}
	return true
}

type protocolFailure struct {
	Status        int
	Code, Message string
	Retryable     bool
}

func (failure protocolFailure) Error() string { return failure.Message }

func classifyProtocolValidation(err error) error {
	message := err.Error()
	switch {
	case strings.Contains(message, "unsupported spec_version"):
		return protocolFailure{Status: http.StatusBadRequest, Code: "unsupported_version", Message: message}
	case strings.Contains(message, "unsupported event type"):
		return protocolFailure{Status: http.StatusBadRequest, Code: "unsupported_event_type", Message: message}
	default:
		return protocolFailure{Status: http.StatusBadRequest, Code: "invalid_message", Message: message}
	}
}

func writeProtocolError(w http.ResponseWriter, err error) {
	var failure protocolFailure
	if errors.As(err, &failure) {
		writeProtocolErrorCode(w, failure.Status, failure.Code, failure.Message, failure.Retryable)
		return
	}
	writeProtocolErrorCode(w, http.StatusBadRequest, "invalid_message", err.Error(), false)
}
func writeProtocolErrorCode(w http.ResponseWriter, status int, code, message string, retryable bool) {
	writeJSON(w, status, epistemic.Error{SpecVersion: epistemic.Version, Code: code, Message: message, Retryable: retryable})
}
