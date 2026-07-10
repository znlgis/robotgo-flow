package serve

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"sync"

	"robotgo-flow/internal/action"
	"robotgo-flow/internal/config"
	"robotgo-flow/internal/executor"
)

// Event 表示写入 stdout 的单条 JSON 事件。
type Event struct {
	Type            string      `json:"type"`
	OK              *bool       `json:"ok,omitempty"`
	Name            string      `json:"name,omitempty"`
	TotalSteps      int         `json:"total_steps,omitempty"`
	Inputs          []InputInfo `json:"inputs,omitempty"`
	Idx             int         `json:"idx,omitempty"`
	StepIdx         int         `json:"step_idx,omitempty"`
	Total           int         `json:"total,omitempty"`
	Action          string      `json:"action,omitempty"`
	Detail          string      `json:"detail,omitempty"`
	Level           string      `json:"level,omitempty"`
	Message         string      `json:"message,omitempty"`
	Error           string      `json:"error,omitempty"`
	EstimatedSec    float64     `json:"estimated_sec,omitempty"`
	ScreenshotPath  string      `json:"screenshot_path,omitempty"`
	TotalElapsedSec float64     `json:"total_elapsed_sec,omitempty"`
}

// InputInfo 对应 config.InputSpec, 用于 loaded 事件。
type InputInfo struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	Required    bool   `json:"required"`
	Mask        bool   `json:"mask"`
}

// Command 表示从 stdin 接收的 JSON 命令。
type Command struct {
	Type   string            `json:"type"`
	Values map[string]string `json:"values,omitempty"`
}

// serveCallback 实现 executor.ProgressCallback, 将 JSON 事件写入 stdout。
type serveCallback struct {
	mu  sync.Mutex
	enc *json.Encoder
}

type serveEngine interface {
	action.Engine
	EnableHuman(enabled bool, speed, mistakeRate float64, idle, scroll, overshoot bool)
	Close()
}

type serveRunner interface {
	SetCallback(cb executor.ProgressCallback)
	WithContext(ctx context.Context)
	EnableDebugScreenshots(enabled bool)
	Run() error
	RunFromStep(startStep int) error
}

type executorRunner struct {
	exe *executor.Executor
}

func (r *executorRunner) SetCallback(cb executor.ProgressCallback) {
	r.exe.SetCallback(cb)
}

func (r *executorRunner) WithContext(ctx context.Context) {
	r.exe.WithContext(ctx)
}

func (r *executorRunner) EnableDebugScreenshots(enabled bool) {
	r.exe.EnableDebugScreenshots(enabled)
}

func (r *executorRunner) Run() error {
	return r.exe.Run()
}

func (r *executorRunner) RunFromStep(startStep int) error {
	return r.exe.RunFromStep(startStep)
}

var (
	loadWorkflow   = config.Load
	newServeRunner = func(eng action.Engine, cfg *config.Workflow, workflowPath string) serveRunner {
		return &executorRunner{exe: executor.New(eng, cfg, workflowPath)}
	}
)

func (s *serveCallback) OnWorkflowStart(name string, totalSteps int) {
	s.mu.Lock()
	s.enc.Encode(Event{Type: "workflow_start", Name: name, TotalSteps: totalSteps})
	s.mu.Unlock()
}

func (s *serveCallback) OnStepStart(stepIdx, totalSteps int, stepName string, estimatedSec float64) {
	s.mu.Lock()
	s.enc.Encode(Event{Type: "step_start", Idx: stepIdx, Total: totalSteps, Name: stepName, EstimatedSec: estimatedSec})
	s.mu.Unlock()
}

func (s *serveCallback) OnActionStart(stepIdx, actionIdx int, actionType, desc string) {
	s.mu.Lock()
	s.enc.Encode(Event{Type: "action_start", StepIdx: stepIdx, Idx: actionIdx, Action: actionType, Detail: desc})
	s.mu.Unlock()
}

func (s *serveCallback) OnActionDone(stepIdx, actionIdx int) {
	s.mu.Lock()
	s.enc.Encode(Event{Type: "action_done", StepIdx: stepIdx, Idx: actionIdx})
	s.mu.Unlock()
}

func (s *serveCallback) OnStepDone(stepIdx int, err error, screenshotPath string) {
	s.mu.Lock()
	ev := Event{Type: "step_done", Idx: stepIdx, ScreenshotPath: screenshotPath}
	if err != nil {
		ev.Error = err.Error()
	}
	s.enc.Encode(ev)
	s.mu.Unlock()
}

func (s *serveCallback) OnWorkflowDone(name string, totalSteps int, err error, totalElapsedSec float64) {
	s.mu.Lock()
	ok := err == nil
	ev := Event{Type: "workflow_done", OK: &ok, Name: name, TotalSteps: totalSteps, TotalElapsedSec: totalElapsedSec}
	if err != nil {
		ev.Error = err.Error()
	}
	s.enc.Encode(ev)
	s.mu.Unlock()
}

func (s *serveCallback) OnLog(level executor.LogLevel, message string, stepIdx int) {
	s.mu.Lock()
	lvlStr := "info"
	switch level {
	case executor.LogWarn:
		lvlStr = "warn"
	case executor.LogError:
		lvlStr = "error"
	}
	s.enc.Encode(Event{Type: "log", Level: lvlStr, Message: message, StepIdx: stepIdx})
	s.mu.Unlock()
}

// emitStopped 发送 "stopped" 事件, 可从 stdin 读取协程安全并发调用。
func (s *serveCallback) emitStopped() {
	s.mu.Lock()
	s.enc.Encode(Event{Type: "stopped"})
	s.mu.Unlock()
}

// inputInfos 将 config.InputSpec 切片转换为 serve InputInfo 切片。
func inputInfos(inputs []config.InputSpec) []InputInfo {
	out := make([]InputInfo, len(inputs))
	for i, inp := range inputs {
		out[i] = InputInfo{
			Name:        inp.Name,
			Label:       inp.Label,
			Placeholder: inp.Placeholder,
			Required:    inp.Required,
			Mask:        inp.Mask,
		}
	}
	return out
}

// Run 在 stdin/stdout 上启动指定工作流的 serve 协议。
// 阻塞直到执行完成或被停止。
//
// 调用方必须在 Run() 返回后关闭 stdin 以防止协程泄漏。
//
// 协议 (JSON Lines, 每行一个对象):
//
//	Go → stdout: {"type":"loaded","ok":true,...}
//	若存在输入, Go 阻塞等待 stdin:
//	调用方 → stdin: {"type":"set_inputs","values":{...}}
//	Go → stdout: {"type":"workflow_start",...}
//	Go → stdout: {"type":"step_start",...}  {"type":"action_start",...}  ...
//	Go → stdout: {"type":"workflow_done","ok":true}
//	调用方 → stdin: {"type":"stop"}  (随时可用)
//	Go → stdout: {"type":"stopped"}
func Run(workflowPath string, stdin io.Reader, stdout io.Writer, fromStep int, debug bool) error {
	// 使用 io.Pipe 包装 stdin，确保 Run 返回时通过关闭 pr 终止 stdin 读取协程，防止协程泄漏
	pr, pw := io.Pipe()
	go func() {
		_, _ = io.Copy(pw, stdin)
		pw.Close()
	}()

	enc := json.NewEncoder(stdout)
	dec := json.NewDecoder(pr)
	defer pr.Close() // Run 返回时强制结束 stdin 读取协程
	cb := &serveCallback{enc: enc}

	// 1. 加载工作流配置
	cfg, err := loadWorkflow(workflowPath)
	if err != nil {
		enc.Encode(Event{Type: "error", Message: err.Error()})
		return err
	}

	// 2. 发送 loaded 事件及输入元数据
	okVal := true
	enc.Encode(Event{
		Type:       "loaded",
		OK:         &okVal,
		Name:       cfg.Name,
		TotalSteps: len(cfg.Steps),
		Inputs:     inputInfos(cfg.Inputs),
	})

	// 3. 若声明了输入, 阻塞等待 set_inputs 命令
	if len(cfg.Inputs) > 0 {
		var cmd Command
		if err := dec.Decode(&cmd); err != nil {
			msg := fmt.Sprintf("读取输入失败: %v", err)
			enc.Encode(Event{Type: "error", Message: msg})
			return fmt.Errorf("读取输入失败: %w", err)
		}
		if cmd.Type != "set_inputs" {
			enc.Encode(Event{Type: "error", Message: fmt.Sprintf("预期 set_inputs 命令, 收到 %q", cmd.Type)})
			return fmt.Errorf("预期 set_inputs 命令, 收到 %q", cmd.Type)
		}
		// 直接解析占位符, 无需创建 executor。
		// ResolveInputs 仅访问 cfg, 因此不需要 engine。
		executor.ResolveInputs(cfg, cmd.Values)
	}

	// 4. 创建可取消的 context 以支持停止
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 5. 启动 stdin 读取 goroutine 以接收 stop 命令
	// 使用 done 通道确保 Run 返回时能干净地关闭。
	done := make(chan struct{})
	defer close(done)

	go func() {
		for {
			// 在阻塞读取 stdin 前先检查退出信号
			select {
			case <-done:
				return
			case <-ctx.Done():
				return
			default:
			}

			var cmd Command
			if err := dec.Decode(&cmd); err != nil {
				return // stdin 已关闭或解析出错
			}
			if cmd.Type == "stop" {
				cancel()
				cb.emitStopped()
				return
			}
		}
	}()

	// 6. 创建引擎和执行器
	workDir := filepath.Dir(workflowPath)
	eng := newServeEngine(workDir)
	defer eng.Close()
	if cfg.Settings.Human.Enabled {
		eng.EnableHuman(true, cfg.Settings.Human.Speed, cfg.Settings.Human.MistakeRate, true, true, false)
	}
	exe := newServeRunner(eng, cfg, workflowPath)
	exe.SetCallback(cb)
	exe.WithContext(ctx)
	exe.EnableDebugScreenshots(debug)

	// 7. 执行工作流
	if fromStep < 1 {
		return fmt.Errorf("起始步骤必须 >= 1，当前值为 %d", fromStep)
	}
	var runErr error
	if fromStep > 1 {
		runErr = exe.RunFromStep(fromStep)
	} else {
		runErr = exe.Run()
	}
	return runErr
}
