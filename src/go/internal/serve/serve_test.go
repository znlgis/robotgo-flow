package serve

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"robotgo-flow/internal/action"
	"robotgo-flow/internal/config"
	"robotgo-flow/internal/executor"
)

func TestEvent_MarshalUnmarshal(t *testing.T) {
	ok := true
	ev := Event{
		Type:       "workflow_done",
		OK:         &ok,
		Name:       "test-workflow",
		TotalSteps: 3,
	}

	data, err := json.Marshal(ev)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Type != ev.Type {
		t.Errorf("type: got %q, want %q", decoded.Type, ev.Type)
	}
	if decoded.Name != ev.Name {
		t.Errorf("name: got %q, want %q", decoded.Name, ev.Name)
	}
	if decoded.TotalSteps != ev.TotalSteps {
		t.Errorf("total_steps: got %d, want %d", decoded.TotalSteps, ev.TotalSteps)
	}
}

func TestEvent_AllTypes(t *testing.T) {
	types := []string{"loaded", "workflow_start", "step_start", "action_start", "action_done",
		"step_done", "workflow_done", "log", "stopped", "error"}
	for _, typ := range types {
		ev := Event{Type: typ}
		data, err := json.Marshal(ev)
		if err != nil {
			t.Errorf("marshal event type %q: %v", typ, err)
			continue
		}
		var decoded Event
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Errorf("unmarshal event type %q: %v", typ, err)
		}
		if decoded.Type != typ {
			t.Errorf("event type %q: got %q", typ, decoded.Type)
		}
	}
}

func TestCommand_MarshalUnmarshal(t *testing.T) {
	cmd := Command{
		Type:   "set_inputs",
		Values: map[string]string{"username": "admin", "password": "secret"},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded Command
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Type != cmd.Type {
		t.Errorf("type: got %q, want %q", decoded.Type, cmd.Type)
	}
	if decoded.Values["username"] != "admin" {
		t.Errorf("values[username]: got %q, want %q", decoded.Values["username"], "admin")
	}
}

func TestInputInfos(t *testing.T) {
	inputs := []config.InputSpec{
		{Name: "username", Label: "用户名", Placeholder: "请输入用户名", Required: true, Mask: false},
		{Name: "password", Label: "密码", Placeholder: "请输入密码", Required: true, Mask: true},
	}

	result := inputInfos(inputs)

	if len(result) != 2 {
		t.Fatalf("expected 2 inputs, got %d", len(result))
	}

	if result[0].Name != "username" {
		t.Errorf("input 0 name: got %q, want %q", result[0].Name, "username")
	}
	if result[0].Mask {
		t.Error("input 0 should not be masked")
	}
	if !result[1].Mask {
		t.Error("input 1 should be masked")
	}
}

func TestInputInfos_Empty(t *testing.T) {
	result := inputInfos(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 inputs, got %d", len(result))
	}

	result = inputInfos([]config.InputSpec{})
	if len(result) != 0 {
		t.Errorf("expected 0 inputs, got %d", len(result))
	}
}

func TestServeCallback_OnLog(t *testing.T) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	cb := &serveCallback{enc: enc}

	cb.OnLog(executor.LogInfo, "test message", 0)

	var ev Event
	if err := json.Unmarshal(buf.Bytes(), &ev); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if ev.Type != "log" {
		t.Errorf("type: got %q, want %q", ev.Type, "log")
	}
	if ev.Level != "info" {
		t.Errorf("level: got %q, want %q", ev.Level, "info")
	}
	if ev.Message != "test message" {
		t.Errorf("message: got %q, want %q", ev.Message, "test message")
	}
}

func TestServeCallback_OnLog_Levels(t *testing.T) {
	tests := []struct {
		level     executor.LogLevel
		wantLevel string
	}{
		{level: executor.LogInfo, wantLevel: "info"},
		{level: executor.LogWarn, wantLevel: "warn"},
		{level: executor.LogError, wantLevel: "error"},
	}

	for _, tt := range tests {
		t.Run(tt.wantLevel, func(t *testing.T) {
			var buf bytes.Buffer
			enc := json.NewEncoder(&buf)
			cb := &serveCallback{enc: enc}

			cb.OnLog(tt.level, "msg", 0)

			var ev Event
			json.Unmarshal(buf.Bytes(), &ev)
			if ev.Level != tt.wantLevel {
				t.Errorf("level: got %q, want %q", ev.Level, tt.wantLevel)
			}
		})
	}
}

func TestServeCallback_ThreadSafety(t *testing.T) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	cb := &serveCallback{enc: enc}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			cb.OnLog(executor.LogInfo, "concurrent log", 0)
			cb.OnActionStart(0, idx, "click", "test")
			cb.OnActionDone(0, idx)
		}(i)
	}
	wg.Wait()

	// 到达此处说明互斥锁正常，没有并发问题
	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	t.Logf("produced %d JSON lines", len(lines))
	if len(lines) < 3 {
		t.Errorf("expected at least 3 lines, got %d", len(lines))
	}
}

func TestServeCallback_OnWorkflowDone_Success(t *testing.T) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	cb := &serveCallback{enc: enc}

	cb.OnWorkflowDone("test-workflow", 3, nil, 0)

	var ev Event
	json.Unmarshal(buf.Bytes(), &ev)

	if ev.Type != "workflow_done" {
		t.Errorf("type: got %q, want %q", ev.Type, "workflow_done")
	}
	if ev.OK == nil || !*ev.OK {
		t.Error("expected ok=true for successful workflow")
	}
	if ev.Error != "" {
		t.Errorf("expected no error, got %q", ev.Error)
	}
	if ev.Name != "test-workflow" {
		t.Errorf("expected workflow name, got %q", ev.Name)
	}
	if ev.TotalSteps != 3 {
		t.Errorf("expected total steps 3, got %d", ev.TotalSteps)
	}
}

type stubServeEngine struct {
	action.MockEngine
	humanEnabled bool
	closed       bool
}

func (s *stubServeEngine) EnableHuman(enabled bool, speed, mistakeRate float64, idle, scroll, overshoot bool) {
	s.humanEnabled = enabled
}

func (s *stubServeEngine) Close() {
	s.closed = true
}

type fakeRunner struct {
	callback executor.ProgressCallback
	ctx      context.Context
	debug    bool
	run      func(*fakeRunner) error
	runFrom  func(*fakeRunner, int) error
}

func (r *fakeRunner) SetCallback(cb executor.ProgressCallback) {
	r.callback = cb
}

func (r *fakeRunner) WithContext(ctx context.Context) {
	r.ctx = ctx
}

func (r *fakeRunner) EnableDebugScreenshots(enabled bool) {
	r.debug = enabled
}

func (r *fakeRunner) Run() error {
	if r.run != nil {
		return r.run(r)
	}
	return nil
}

func (r *fakeRunner) RunFromStep(startStep int) error {
	if r.runFrom != nil {
		return r.runFrom(r, startStep)
	}
	return r.Run()
}

func Test执行_解析输入并应用选项(t *testing.T) {
	oldLoadWorkflow := loadWorkflow
	oldNewServeEngine := newServeEngine
	oldNewServeRunner := newServeRunner
	defer func() {
		loadWorkflow = oldLoadWorkflow
		newServeEngine = oldNewServeEngine
		newServeRunner = oldNewServeRunner
	}()

	cfg := &config.Workflow{
		Name: "input-workflow",
		Inputs: []config.InputSpec{
			{Name: "username", Label: "用户名", Required: true},
		},
		Settings: config.Settings{
			Human: config.HumanConf{Enabled: true, Speed: 1.2, MistakeRate: 0.1},
		},
		Steps: []config.Step{
			{Name: "step1", Actions: []config.Action{{Type: &config.TypeSpec{Into: "input.png", Text: "hello $input.username"}}}},
		},
	}

	engineStub := &stubServeEngine{}
	runnerStub := &fakeRunner{}
	loadWorkflow = func(string) (*config.Workflow, error) { return cfg, nil }
	newServeEngine = func(string) serveEngine { return engineStub }
	newServeRunner = func(eng action.Engine, gotCfg *config.Workflow, workflowPath string) serveRunner {
		if gotCfg != cfg {
			t.Fatalf("expected original workflow pointer to be passed through")
		}
		return runnerStub
	}

	var out bytes.Buffer
	in := strings.NewReader("{\"type\":\"set_inputs\",\"values\":{\"username\":\"alice\"}}\n")
	if err := Run("/tmp/workflow.yaml", in, &out, 1, true); err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if got := cfg.Steps[0].Actions[0].Type.Text; got != "hello alice" {
		t.Fatalf("expected input placeholder to be resolved, got %q", got)
	}
	if !engineStub.humanEnabled {
		t.Fatal("expected serve engine human mode to be enabled")
	}
	if !engineStub.closed {
		t.Fatal("expected serve engine to be closed")
	}
	if !runnerStub.debug {
		t.Fatal("expected debug screenshots flag to propagate to runner")
	}
	if !strings.Contains(out.String(), "\"type\":\"loaded\"") {
		t.Fatalf("expected loaded event, got %s", out.String())
	}
}

func Test执行_停止命令取消执行(t *testing.T) {
	oldLoadWorkflow := loadWorkflow
	oldNewServeEngine := newServeEngine
	oldNewServeRunner := newServeRunner
	defer func() {
		loadWorkflow = oldLoadWorkflow
		newServeEngine = oldNewServeEngine
		newServeRunner = oldNewServeRunner
	}()

	cfg := &config.Workflow{
		Name:     "stop-workflow",
		Settings: config.Settings{},
		Steps:    []config.Step{{Name: "step1", Actions: []config.Action{{Sleep: 0.01}}}},
	}

	loadWorkflow = func(string) (*config.Workflow, error) { return cfg, nil }
	newServeEngine = func(string) serveEngine { return &stubServeEngine{} }
	newServeRunner = func(eng action.Engine, gotCfg *config.Workflow, workflowPath string) serveRunner {
		return &fakeRunner{
			run: func(r *fakeRunner) error {
				<-r.ctx.Done()
				return executor.ErrCancelled
			},
		}
	}

	var out bytes.Buffer
	err := Run("/tmp/workflow.yaml", strings.NewReader("{\"type\":\"stop\"}\n"), &out, 1, false)
	if !errors.Is(err, executor.ErrCancelled) {
		t.Fatalf("expected cancellation error, got %v", err)
	}

	// emitStopped() 由 stdin goroutine 异步写入，轮询等待最多 1 秒
	deadline := time.Now().Add(time.Second)
	for {
		if strings.Contains(out.String(), `"type":"stopped"`) {
			break
		}
		if time.Now().After(deadline) {
			t.Error("stopped 事件未出现")
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func Test执行_无效输入命令返回错误事件(t *testing.T) {
	oldLoadWorkflow := loadWorkflow
	oldNewServeEngine := newServeEngine
	oldNewServeRunner := newServeRunner
	defer func() {
		loadWorkflow = oldLoadWorkflow
		newServeEngine = oldNewServeEngine
		newServeRunner = oldNewServeRunner
	}()

	cfg := &config.Workflow{
		Name: "invalid-input-workflow",
		Inputs: []config.InputSpec{
			{Name: "username", Label: "用户名", Required: true},
		},
		Steps: []config.Step{{Name: "step1", Actions: []config.Action{{Sleep: 0.01}}}},
	}

	loadWorkflow = func(string) (*config.Workflow, error) { return cfg, nil }
	newServeEngine = func(string) serveEngine { return &stubServeEngine{} }
	newServeRunner = func(eng action.Engine, gotCfg *config.Workflow, workflowPath string) serveRunner {
		t.Fatal("runner should not be created when set_inputs command is invalid")
		return nil
	}

	var out bytes.Buffer
	err := Run("/tmp/workflow.yaml", strings.NewReader("{\"type\":\"stop\"}\n"), &out, 1, false)
	if err == nil || !strings.Contains(err.Error(), "预期 set_inputs 命令") {
		t.Fatalf("expected set_inputs validation error, got %v", err)
	}
	if !strings.Contains(out.String(), "\"type\":\"error\"") {
		t.Fatalf("expected error event, got %s", out.String())
	}
}
