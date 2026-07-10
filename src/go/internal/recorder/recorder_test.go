package recorder

import (
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	r := New("output.yaml", "templates", strings.NewReader(""))
	if r == nil {
		t.Fatal("New returned nil")
	}
	if r.templateDir != "templates" {
		t.Errorf("templateDir: got %q, want %q", r.templateDir, "templates")
	}
	if r.outputPath != "output.yaml" {
		t.Errorf("outputPath: got %q, want %q", r.outputPath, "output.yaml")
	}
	if r.wf == nil {
		t.Fatal("workflow should not be nil")
	}
	if r.wf.Settings.ElementTimeout != 10 {
		t.Errorf("ElementTimeout: got %d, want 10", r.wf.Settings.ElementTimeout)
	}
}

func TestNew_DefaultSettings(t *testing.T) {
	r := New("test.yaml", "tmpl", strings.NewReader(""))

	if r.wf.Name != "" {
		t.Errorf("name should be empty initially, got %q", r.wf.Name)
	}
	if len(r.wf.Steps) != 0 {
		t.Errorf("steps should be empty initially, got %d", len(r.wf.Steps))
	}
}

func TestRecorder_SaveStructure(t *testing.T) {
	// Test that the workflow structure is properly initialized
	r := New("test.yaml", "tmpl", strings.NewReader(""))

	// Verify the workflow type
	if r.wf == nil {
		t.Fatal("workflow is nil")
	}

	// Check settings defaults
	cfg := r.wf
	if cfg.Settings.ElementTimeout != 10 {
		t.Errorf("expected ElementTimeout=10, got %d", cfg.Settings.ElementTimeout)
	}
}
