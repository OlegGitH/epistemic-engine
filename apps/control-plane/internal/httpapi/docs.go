package httpapi

import (
	"net/http"

	apiassets "github.com/OlegGitH/epistemic-engine/apps/control-plane/api"
)

const swaggerHTML = `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Epistemic Engine API</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5/swagger-ui.css">
  <style>body { margin: 0; background: #fafafa; }</style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5/swagger-ui-bundle.js"></script>
  <script>
    window.onload = function () {
      SwaggerUIBundle({
        url: "/openapi.yaml",
        dom_id: "#swagger-ui",
        deepLinking: true,
        displayRequestDuration: true,
        filter: true,
        tryItOutEnabled: true
      });
    };
  </script>
</body>
</html>
`

var documentationAssets = map[string]struct {
	name        string
	contentType string
}{
	"/openapi.yaml":                              {"openapi/epistemic-control-plane.yaml", "application/yaml; charset=utf-8"},
	"/schemas/trace-event.schema.json":           {"schemas/trace-event.schema.json", "application/schema+json"},
	"/schemas/decision-certificate.schema.json": {"schemas/decision-certificate.schema.json", "application/schema+json"},
}

func serveDocumentation(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/" || r.URL.Path == "/docs" {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowed(w)
			return true
		}
		http.Redirect(w, r, "/docs/", http.StatusTemporaryRedirect)
		return true
	}
	if r.URL.Path == "/docs/" {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowed(w)
			return true
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; connect-src 'self'; img-src data: https:; style-src 'self' 'unsafe-inline' https://unpkg.com; script-src 'unsafe-inline' https://unpkg.com")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(swaggerHTML))
		return true
	}
	asset, found := documentationAssets[r.URL.Path]
	if !found {
		return false
	}
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		methodNotAllowed(w)
		return true
	}
	content, err := apiassets.Read(asset.name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "API documentation is unavailable"})
		return true
	}
	w.Header().Set("Content-Type", asset.contentType)
	w.Header().Set("Cache-Control", "public, max-age=300")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
	return true
}

func methodNotAllowed(w http.ResponseWriter) {
	w.Header().Set("Allow", "GET, HEAD")
	writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
}
