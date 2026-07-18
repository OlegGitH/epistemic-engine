// Package api embeds the public API contract so the control-plane binary can
// serve documentation without depending on repository files at runtime.
package api

import "embed"

//go:embed openapi/epistemic-control-plane.yaml schemas/*.json
var files embed.FS

// Read returns an embedded API asset.
func Read(name string) ([]byte, error) {
	return files.ReadFile(name)
}
