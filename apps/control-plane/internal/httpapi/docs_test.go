package httpapi

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/analysis"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/service"
	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/store"
)

func TestDocumentationRoutes(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()))
	tests := []struct {
		path        string
		contentType string
		contains    string
	}{
		{path: "/docs/", contentType: "text/html", contains: "SwaggerUIBundle"},
		{path: "/openapi.yaml", contentType: "application/yaml", contains: "openapi: 3.1.0"},
		{path: "/schemas/trace-event.schema.json", contentType: "application/schema+json", contains: `"$schema"`},
		{path: "/schemas/decision-certificate.schema.json", contentType: "application/schema+json", contains: `"$schema"`},
	}
	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, test.path, nil))
			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
			if !strings.HasPrefix(response.Header().Get("Content-Type"), test.contentType) {
				t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
			}
			if !strings.Contains(response.Body.String(), test.contains) {
				t.Fatalf("response does not contain %q", test.contains)
			}
		})
	}
}

func TestDocumentationRootRedirects(t *testing.T) {
	h := New(service.New(store.NewMemory(), analysis.NewRulesAnalyzer()))
	response := httptest.NewRecorder()
	h.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	if response.Code != http.StatusTemporaryRedirect || response.Header().Get("Location") != "/docs/" {
		t.Fatalf("unexpected redirect: status=%d location=%q", response.Code, response.Header().Get("Location"))
	}
}
