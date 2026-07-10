package notify

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestErrorLogEntry_Marshal(t *testing.T) {
	entry := ErrorLogEntry{
		Time:       "2025-01-15T10:30:00Z",
		Step:       "login",
		Action:     1,
		Error:      "element not found",
		Screenshot: "/path/to/screenshot.png",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded ErrorLogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Step != entry.Step {
		t.Errorf("step: got %q, want %q", decoded.Step, entry.Step)
	}
	if decoded.Action != entry.Action {
		t.Errorf("action: got %d, want %d", decoded.Action, entry.Action)
	}
	if decoded.Error != entry.Error {
		t.Errorf("error: got %q, want %q", decoded.Error, entry.Error)
	}
}

func TestLogError(t *testing.T) {
	dir := t.TempDir()

	LogError(dir, "login-step", 2, os.ErrNotExist, "/tmp/ss.png")

	// Verify file was created
	logPath := filepath.Join(dir, "workflow-error.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	var entry ErrorLogEntry
	dec := json.NewDecoder(bytes.NewReader(data))
	if err := dec.Decode(&entry); err != nil {
		t.Fatalf("decode log entry: %v", err)
	}

	if entry.Step != "login-step" {
		t.Errorf("step: got %q, want %q", entry.Step, "login-step")
	}
	if entry.Action != 2 {
		t.Errorf("action: got %d, want 2", entry.Action)
	}
	if entry.Error != os.ErrNotExist.Error() {
		t.Errorf("error: got %q, want %q", entry.Error, os.ErrNotExist.Error())
	}
	if entry.Screenshot != "/tmp/ss.png" {
		t.Errorf("screenshot: got %q, want %q", entry.Screenshot, "/tmp/ss.png")
	}
}

func TestLogError_Appends(t *testing.T) {
	dir := t.TempDir()

	LogError(dir, "step1", 0, os.ErrNotExist, "/s1.png")
	LogError(dir, "step2", 1, os.ErrPermission, "/s2.png")

	logPath := filepath.Join(dir, "workflow-error.log")
	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	content := string(data)

	// Verify both entries were written
	if !strings.Contains(content, "\"step\": \"step1\"") {
		t.Error("missing step1 entry in log")
	}
	if !strings.Contains(content, "\"step\": \"step2\"") {
		t.Error("missing step2 entry in log")
	}
	if !strings.Contains(content, os.ErrNotExist.Error()) {
		t.Error("missing ErrNotExist in log")
	}
	if !strings.Contains(content, os.ErrPermission.Error()) {
		t.Error("missing ErrPermission in log")
	}
}

func TestPrintError(t *testing.T) {
	// Redirect stderr to capture output
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	PrintError("TestTitle", "Test Error Message")

	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	if !strings.Contains(output, "TestTitle") {
		t.Errorf("expected title in output, got: %s", output)
	}
	if !strings.Contains(output, "Test Error Message") {
		t.Errorf("expected message in output, got: %s", output)
	}
}

func TestErrorLogEntry_AllFields(t *testing.T) {
	entry := ErrorLogEntry{
		Time:       "2025-06-25T12:00:00+08:00",
		Step:       "fill-form",
		Action:     3,
		Error:      "timeout waiting for element",
		Screenshot: "C:\\screenshots\\err_001.png",
	}

	data, _ := json.Marshal(entry)
	var decoded map[string]interface{}
	json.Unmarshal(data, &decoded)

	expectedKeys := []string{"time", "step", "action", "error", "screenshot"}
	for _, key := range expectedKeys {
		if _, ok := decoded[key]; !ok {
			t.Errorf("missing key %q in JSON output", key)
		}
	}
}
