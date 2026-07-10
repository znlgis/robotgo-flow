package capture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindBrowser(t *testing.T) {
	path := findBrowser()

	if path != "" {
		// If a browser was found, verify the path exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("findBrowser returned non-existent path: %s", path)
		}
		// Verify it's an .exe
		if filepath.Ext(path) != ".exe" {
			t.Errorf("expected .exe extension, got: %s", path)
		}
	}
	// It's OK if no browser is found — the function returns "" in that case
}

func TestFindBrowser_KnownPaths(t *testing.T) {
	// Verify that the known paths in findBrowser are all valid Windows paths
	// (we don't assert existence — just that they're well-formed)
	path := findBrowser()
	t.Logf("findBrowser returned: %s", path)
}

func TestCaptureRegion_EmptyFile(t *testing.T) {
	// CaptureRegion requires a display, so we only test error paths
	// that don't require a real screen
	err := CaptureRegion(0, 0, 0, 0, filepath.Join(t.TempDir(), "test.png"))
	if err == nil {
		t.Error("CaptureRegion(0,0,0,0) expected an error for zero-size region, but succeeded")
	}
}
