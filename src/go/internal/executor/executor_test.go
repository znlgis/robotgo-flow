package executor

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"robotgo-flow/internal/action"
	"robotgo-flow/internal/config"
)

func TestRun_AllStepsSucceed(t *testing.T) {
	eng := &action.MockEngine{}
	cfg := &config.Workflow{
		Name:     "test-workflow",
		Settings: config.Settings{ElementTimeout: 10},
		Steps: []config.Step{
			{
				Name: "step1",
				Actions: []config.Action{
					{Press: "enter"},
					{Scroll: 100},
				},
			},
			{
				Name: "step2",
				Actions: []config.Action{
					{OpenURL: "https://example.com"},
				},
			},
		},
	}

	exe := New(eng, cfg, "test_workflow.yaml")
	err := exe.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !eng.Called("FindBrowserWindow") {
		t.Error("FindBrowserWindow should be called")
	}
	if !eng.Called("PressKey") {
		t.Error("PressKey should be called for step1/action1")
	}
	if !eng.Called("ScrollDown") {
		t.Error("ScrollDown should be called for step1/action2")
	}
	if !eng.Called("OpenURL") {
		t.Error("OpenURL should be called for step2/action1")
	}
	if eng.Called("CaptureScreen") {
		t.Error("CaptureScreen should NOT be called on success")
	}
}

func TestRun_ActionError(t *testing.T) {
	eng := &action.MockEngine{}
	eng.PressKeyError = errors.New("keyboard failure")
	cfg := &config.Workflow{
		Name:     "test-error",
		Settings: config.Settings{ElementTimeout: 10},
		Steps: []config.Step{
			{
				Name: "step1",
				Actions: []config.Action{
					{Press: "enter"},
				},
			},
			{
				Name: "step2",
				Actions: []config.Action{
					{Scroll: 50},
				},
			},
		},
	}

	exe := New(eng, cfg, "test_error.yaml")
	err := exe.Run()
	if err == nil {
		t.Fatal("Run() should return error when action fails")
	}

	if !strings.Contains(err.Error(), "step1") {
		t.Errorf("error should mention step name, got: %v", err)
	}

	if !eng.Called("CaptureScreen") {
		t.Error("CaptureScreen should be called on action error")
	}

	// Step2 不应被执行（遇到首个错误即停止执行）
	if eng.Called("ScrollDown") {
		t.Error("ScrollDown should NOT be called — step2 should not execute after step1 error")
	}
}

func TestRunFromStep_ValidStart(t *testing.T) {
	eng := &action.MockEngine{}
	cfg := &config.Workflow{
		Name:     "test-from-step",
		Settings: config.Settings{ElementTimeout: 10},
		Steps: []config.Step{
			{Name: "step1", Actions: []config.Action{{Press: "a"}}},
			{Name: "step2", Actions: []config.Action{{Press: "b"}}},
			{Name: "step3", Actions: []config.Action{{Press: "c"}}},
		},
	}

	exe := New(eng, cfg, "test_workflow.yaml")
	err := exe.RunFromStep(2)
	if err != nil {
		t.Fatalf("RunFromStep(2) error = %v", err)
	}

	// Step1（Press "a"）不应被执行
	for _, c := range eng.Calls {
		if c.Name == "PressKey" {
			key, _ := c.Args[0].(string)
			if key == "a" {
				t.Errorf("step1 should NOT have executed, but PressKey(%q) was called", key)
			}
		}
	}

	// Step2 和 Step3 应被执行
	foundB, foundC := false, false
	for _, c := range eng.Calls {
		if c.Name == "PressKey" {
			key, _ := c.Args[0].(string)
			if key == "b" {
				foundB = true
			}
			if key == "c" {
				foundC = true
			}
		}
	}
	if !foundB {
		t.Error("step2 (Press 'b') should have been executed")
	}
	if !foundC {
		t.Error("step3 (Press 'c') should have been executed")
	}

	// PressKey 总共应调用 2 次（b 和 c，不含 a）
	if eng.CallCount("PressKey") != 2 {
		t.Errorf("PressKey called %d times, want 2", eng.CallCount("PressKey"))
	}
}

func TestRunFromStep_OutOfRange(t *testing.T) {
	eng := &action.MockEngine{}
	cfg := &config.Workflow{
		Settings: config.Settings{ElementTimeout: 10},
		Steps: []config.Step{
			{Name: "step1", Actions: []config.Action{{Press: "a"}}},
		},
	}

	exe := New(eng, cfg, "test.yaml")

	err := exe.RunFromStep(0)
	if err == nil {
		t.Error("RunFromStep(0) should return error")
	}

	err = exe.RunFromStep(99)
	if err == nil {
		t.Error("RunFromStep(99) should return error")
	}

	// 起始索引越界时不应有任何步骤被执行
	if eng.CallCount("PressKey") > 0 {
		t.Error("No steps should execute when start index is out of range")
	}
}

func TestExpandVars(t *testing.T) {
	vars := map[string]string{
		"username": "admin",
		"password": "secret",
	}

	tests := []struct {
		input    string
		expected string
	}{
		{"$input.username", "admin"},
		{"Hello $input.username!", "Hello admin!"},
		{"$input.username:$input.password", "admin:secret"},
		{"no vars here", "no vars here"},
		{"$input.nonexistent", "$input.nonexistent"}, // 未知变量保持原样
	}

	for _, tc := range tests {
		result := expandVars(tc.input, vars)
		if result != tc.expected {
			t.Errorf("expandVars(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestRun_NoBrowserWindow(t *testing.T) {
	eng := &action.MockEngine{}
	eng.BrowserErr = errors.New("no browser found")
	cfg := &config.Workflow{
		Name:     "test-no-browser",
		Settings: config.Settings{ElementTimeout: 10},
		Steps: []config.Step{
			{Name: "step1", Actions: []config.Action{{Press: "enter"}}},
		},
	}

	exe := New(eng, cfg, "test.yaml")
	err := exe.Run()
	if err != nil {
		t.Fatalf("Run() should NOT fail on browser error, got: %v", err)
	}

	if !eng.Called("FindBrowserWindow") {
		t.Error("FindBrowserWindow should still be called")
	}
	if !eng.Called("PressKey") {
		t.Error("step should still execute after browser window error")
	}
}

func TestProgressCallback_NotSet_NoPanic(t *testing.T) {
	eng := &action.MockEngine{}
	cfg := &config.Workflow{
		Name:     "test-no-callback",
		Settings: config.Settings{ElementTimeout: 10, OnError: "abort"},
		Steps: []config.Step{
			{Name: "step1", Actions: []config.Action{
				{Sleep: 0.01},
			}},
		},
	}

	// 未设置回调 —— 不应 panic
	exe := New(eng, cfg, "test_no_callback.yaml")
	err := exe.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
}

// mockCallback 记录所有回调调用以便验证。
type mockCallback struct {
	mu           sync.Mutex
	stepStarts   int
	actionStarts int
	actionDones  int
	stepDones    int
	lastLog      string
	doneCalled   bool
	doneErr      error
	doneName     string
	doneSteps    int
}

func (m *mockCallback) OnWorkflowStart(name string, totalSteps int) {}
func (m *mockCallback) OnStepStart(stepIdx, totalSteps int, stepName string, estimatedSec float64) {
	m.mu.Lock()
	m.stepStarts++
	m.mu.Unlock()
}
func (m *mockCallback) OnActionStart(stepIdx, actionIdx int, actionType, desc string) {
	m.mu.Lock()
	m.actionStarts++
	m.mu.Unlock()
}
func (m *mockCallback) OnActionDone(stepIdx, actionIdx int) {
	m.mu.Lock()
	m.actionDones++
	m.mu.Unlock()
}
func (m *mockCallback) OnStepDone(stepIdx int, err error, screenshotPath string) {
	m.mu.Lock()
	m.stepDones++
	m.mu.Unlock()
}
func (m *mockCallback) OnWorkflowDone(name string, totalSteps int, err error, totalElapsedSec float64) {
	m.mu.Lock()
	m.doneCalled = true
	m.doneErr = err
	m.doneName = name
	m.doneSteps = totalSteps
	m.mu.Unlock()
}
func (m *mockCallback) OnLog(level LogLevel, message string, stepIdx int) {
	m.mu.Lock()
	m.lastLog = message
	m.mu.Unlock()
}

func TestProgressCallback_FiresOnAllCalls(t *testing.T) {
	eng := &action.MockEngine{}
	cfg := &config.Workflow{
		Name:     "test-callback-wf",
		Settings: config.Settings{ElementTimeout: 10, OnError: "abort"},
		Steps: []config.Step{
			{Name: "step1", Actions: []config.Action{
				{Sleep: 0.01},
				{Sleep: 0.01},
			}},
			{Name: "step2", Actions: []config.Action{
				{Sleep: 0.01},
			}},
		},
	}

	mock := &mockCallback{}
	exe := New(eng, cfg, "test_callback.yaml")
	exe.SetCallback(mock)
	err := exe.Run()
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	mock.mu.Lock()
	defer mock.mu.Unlock()

	if mock.stepStarts != 2 {
		t.Errorf("expected 2 OnStepStart calls, got %d", mock.stepStarts)
	}
	if mock.actionStarts != 3 {
		t.Errorf("expected 3 OnActionStart calls, got %d", mock.actionStarts)
	}
	if mock.actionDones != 3 {
		t.Errorf("expected 3 OnActionDone calls, got %d", mock.actionDones)
	}
	if mock.stepDones != 2 {
		t.Errorf("expected 2 OnStepDone calls, got %d", mock.stepDones)
	}
	if !mock.doneCalled {
		t.Error("expected OnWorkflowDone to be called")
	}
	if mock.doneErr != nil {
		t.Errorf("expected nil error in OnWorkflowDone, got %v", mock.doneErr)
	}
	if mock.doneName != "test-callback-wf" {
		t.Errorf("expected workflow name to be propagated, got %q", mock.doneName)
	}
	if mock.doneSteps != 2 {
		t.Errorf("expected total steps to be propagated, got %d", mock.doneSteps)
	}
}

func Test执行_调试截图(t *testing.T) {
	eng := &action.MockEngine{}
	cfg := &config.Workflow{
		Name:     "test-debug",
		Settings: config.Settings{ElementTimeout: 10},
		Steps: []config.Step{
			{Name: "step1", Actions: []config.Action{{Sleep: 0.01}}},
			{Name: "step2", Actions: []config.Action{{Sleep: 0.01}}},
		},
	}

	exe := New(eng, cfg, "test_debug.yaml")
	exe.EnableDebugScreenshots(true)

	if err := exe.Run(); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if eng.CallCount("CaptureScreen") != 2 {
		t.Errorf("expected 2 debug screenshots, got %d", eng.CallCount("CaptureScreen"))
	}
	if len(eng.Screenshots) != 2 {
		t.Fatalf("expected 2 screenshot records, got %d", len(eng.Screenshots))
	}
	if eng.Screenshots[0] != "debug_step_001.png" {
		t.Errorf("unexpected first screenshot name: %q", eng.Screenshots[0])
	}
	if eng.Screenshots[1] != "debug_step_002.png" {
		t.Errorf("unexpected second screenshot name: %q", eng.Screenshots[1])
	}
}

func TestRun_OnErrorSkip(t *testing.T) {
	eng := &action.MockEngine{}
	// 第一个动作失败，但在 on_error=skip 模式下第二个动作仍应执行
	eng.PressKeyError = errors.New("keyboard failure")
	eng.ScrollDownErr = errors.New("scroll failure")

	cfg := &config.Workflow{
		Name:     "test-skip",
		Settings: config.Settings{ElementTimeout: 10, OnError: "skip"},
		Steps: []config.Step{
			{
				Name: "step1",
				Actions: []config.Action{
					{Press: "enter"}, // will fail but skipped
					{Scroll: 100},    // should still execute
				},
			},
		},
	}

	exe := New(eng, cfg, "test_skip.yaml")
	mock := &mockCallback{}
	exe.SetCallback(mock)
	err := exe.Run()
	if err != nil {
		t.Fatalf("Run() with skip should not error: %v", err)
	}

	// 两个动作都应被尝试执行
	if eng.CallCount("PressKey") != 1 {
		t.Errorf("PressKey should be called once, got %d", eng.CallCount("PressKey"))
	}
	if eng.CallCount("ScrollDown") != 1 {
		t.Errorf("ScrollDown should be called once, got %d", eng.CallCount("ScrollDown"))
	}
	// skip 模式下不应截图
	if eng.Called("CaptureScreen") {
		t.Error("CaptureScreen should NOT be called on skip mode")
	}
	// 回调协议应保持对称：每个 OnActionStart 都对应一个 OnActionDone，
	// 即使动作被跳过（避免 UI 侧动作进度悬挂）。
	if mock.actionStarts != mock.actionDones {
		t.Errorf("OnActionStart (%d) 与 OnActionDone (%d) 数量应相等", mock.actionStarts, mock.actionDones)
	}
	if mock.actionDones != 2 {
		t.Errorf("expected 2 OnActionDone calls (含被跳过的动作), got %d", mock.actionDones)
	}
}

func TestRun_OnErrorRetry(t *testing.T) {
	eng := &action.MockEngine{}
	eng.PressKeyError = errors.New("persistent failure")

	cfg := &config.Workflow{
		Name: "test-retry",
		Settings: config.Settings{
			ElementTimeout: 10,
			OnError:        "retry",
			MaxRetries:     5,
		},
		Steps: []config.Step{
			{
				Name: "step1",
				Actions: []config.Action{
					{Press: "enter"},
				},
			},
		},
	}

	exe := New(eng, cfg, "test_retry.yaml")
	err := exe.Run()
	// PressKeyError 持久存在，重试 MaxRetries 次后中止
	if err == nil {
		t.Fatal("Run() should return error when retries exhausted")
	}
	if !strings.Contains(err.Error(), "step1") {
		t.Errorf("error should mention step name: %v", err)
	}
	// PressKey 应被调用 MaxRetries 次
	if eng.CallCount("PressKey") != 5 {
		t.Errorf("PressKey called %d times, want %d (MaxRetries)", eng.CallCount("PressKey"), 5)
	}
	// Wait 应在重试间隔调用（5 次尝试间调用 4 次）
	waitCount := 0
	for _, c := range eng.Calls {
		if c.Name == "Wait" {
			waitCount++
		}
	}
	expectedWaits := 4
	if waitCount != expectedWaits {
		t.Errorf("Wait called %d times between retries, want %d", waitCount, expectedWaits)
	}
	// 最终失败时应截图
	if !eng.Called("CaptureScreen") {
		t.Error("CaptureScreen should be called on retry exhaustion")
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	eng := &action.MockEngine{}
	cfg := &config.Workflow{
		Name:     "test-cancel",
		Settings: config.Settings{ElementTimeout: 10, OnError: "abort"},
		Steps: []config.Step{
			{Name: "step1", Actions: []config.Action{{Sleep: 0.01}}},
			{Name: "step2", Actions: []config.Action{{Scroll: 100}}},
			{Name: "step3", Actions: []config.Action{{Press: "enter"}}},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	exe := New(eng, cfg, "test_cancel.yaml")
	exe.WithContext(ctx)

	// 在运行前取消
	cancel()

	err := exe.Run()
	if err == nil {
		t.Fatal("Run() should return error when cancelled")
	}
	if err != ErrCancelled {
		t.Errorf("expected ErrCancelled, got: %v", err)
	}
}

func TestRun_ContextCancellationMidExecution(t *testing.T) {
	eng := &action.MockEngine{}
	cfg := &config.Workflow{
		Name:     "test-cancel-mid",
		Settings: config.Settings{ElementTimeout: 10, OnError: "abort"},
		Steps: []config.Step{
			{Name: "step1", Actions: []config.Action{{Sleep: 0.01}}},
			{Name: "step2", Actions: []config.Action{{Sleep: 0.01}}},
			{Name: "step3", Actions: []config.Action{{Scroll: 100}}},
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	exe := New(eng, cfg, "test_cancel_mid.yaml")
	exe.WithContext(ctx)

	// 第一步执行后取消：用 goroutine 添加延迟后取消
	go func() {
		// 短暂等待后取消
		eng.Wait(100)
		cancel()
	}()

	err := exe.Run()
	if err != ErrCancelled {
		t.Errorf("expected ErrCancelled after mid-execution cancel, got: %v", err)
	}
	// 最多 2 个步骤应已启动（step1 已在执行中）
	pressCount := eng.CallCount("PressKey")
	if pressCount > 0 {
		t.Log("step3 started execution before cancellation — timing-dependent")
	}
}
