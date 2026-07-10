package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewEngine(t *testing.T) {
	dir := t.TempDir()
	templateDir := filepath.Join(dir, "templates")
	screenshotDir := filepath.Join(dir, "screenshots")

	e := NewEngine(templateDir, screenshotDir)

	if e == nil {
		t.Fatal("NewEngine returned nil")
	}
	if e.templateDir != templateDir {
		t.Errorf("templateDir: got %q, want %q", e.templateDir, templateDir)
	}
	if e.screenshotDir != screenshotDir {
		t.Errorf("screenshotDir: got %q, want %q", e.screenshotDir, screenshotDir)
	}
	if e.defaultDelay != 500 {
		t.Errorf("defaultDelay: got %d, want 500", e.defaultDelay)
	}

	// 截图目录应被创建
	if _, err := os.Stat(screenshotDir); os.IsNotExist(err) {
		t.Errorf("screenshot dir %q was not created", screenshotDir)
	}
}

func TestNewEngine_Defaults(t *testing.T) {
	e := NewEngine("templates", "screenshots")

	if e.human != nil {
		t.Error("human mode should be nil by default")
	}
}

func TestEnableHuman(t *testing.T) {
	e := NewEngine("t", "s")
	e.EnableHuman(true, 1.0, 0.05, true, true, false)

	if e.human == nil {
		t.Fatal("human mode should be enabled (non-nil)")
	}
}

func TestSwitchTab_Validation(t *testing.T) {
	e := NewEngine("t", "s")

	tests := []struct {
		index   int
		wantErr bool
	}{
		{index: 0, wantErr: true},
		{index: 1, wantErr: false},
		{index: 5, wantErr: false},
		{index: 9, wantErr: false},
		{index: 10, wantErr: true},
		{index: -1, wantErr: true},
	}

	for _, tt := range tests {
		err := e.SwitchTab(tt.index)
		if (err != nil) != tt.wantErr {
			t.Errorf("SwitchTab(%d) error = %v, wantErr = %v", tt.index, err, tt.wantErr)
		}
	}
}
