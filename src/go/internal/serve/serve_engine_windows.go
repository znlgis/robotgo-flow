//go:build windows

package serve

import (
	"path/filepath"

	"robotgo-flow/internal/engine"
)

var newServeEngine = func(workDir string) serveEngine {
	return engine.NewEngine(workDir, filepath.Join(workDir, "screenshots"))
}
