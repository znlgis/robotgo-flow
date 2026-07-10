package main

/*
#include <stdlib.h>

// event_callback_t 是 C# 侧注册的回调函数签名。
typedef void (*event_callback_t)(const char* json);

// callCSharp 从 Go 侧调用 C# 回调。回调指针由 Go 侧管理（atomic.Pointer），
// 作为参数传入，避免 c-shared 模式下 C 全局变量的多重定义问题。
static void callCSharp(event_callback_t cb, const char* json) {
	if (cb != NULL) {
		cb(json);
	}
}
*/
import "C"
import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"unsafe"

	"robotgo-flow/internal/capture"
	"robotgo-flow/internal/config"
	"robotgo-flow/internal/engine"
	"robotgo-flow/internal/executor"
	"robotgo-flow/internal/recorder"
)

func main() {}

// ---------- 生命周期 ----------

//export RobotgoInit
func RobotgoInit() *C.char {
	return C.CString(`{"version":"dev"}`)
}

//export RobotgoDestroy
func RobotgoDestroy() {
	// 预留清理逻辑
}

var gCallback unsafe.Pointer // 原子访问的事件回调指针

//export RobotgoSetCallback
func RobotgoSetCallback(cb unsafe.Pointer) {
	atomic.StorePointer(&gCallback, cb)
}

//export RobotgoFreeString
func RobotgoFreeString(str *C.char) {
	C.free(unsafe.Pointer(str))
}

// ---------- 预加载（仅加载配置，不执行） ----------

//export RobotgoPreload
func RobotgoPreload(workflowPath *C.char) (ret *C.char) {
	defer func() {
		if r := recover(); r != nil {
			ret = C.CString(fmt.Sprintf(`{"ok":false,"error":"内部错误(panic): %v"}`, r))
		}
	}()

	wfPath := C.GoString(workflowPath)
	cfg, err := config.Load(wfPath)
	if err != nil {
		return C.CString(goErrorJSON("加载工作流失败: " + err.Error()))
	}

	inputs := make([]InputInfo, len(cfg.Inputs))
	for i, inp := range cfg.Inputs {
		inputs[i] = InputInfo{
			Name:        inp.Name,
			Label:       inp.Label,
			Placeholder: inp.Placeholder,
			Required:    inp.Required,
			Mask:        inp.Mask,
		}
	}

	result := map[string]interface{}{
		"ok":          true,
		"name":        cfg.Name,
		"total_steps": len(cfg.Steps),
		"inputs":      inputs,
	}
	data, err := json.Marshal(result)
	if err != nil {
		return C.CString(goErrorJSON("序列化失败: " + err.Error()))
	}
	return C.CString(string(data))
}

// ---------- 内部：事件推送 ----------

// emitEvent 将事件序列化为 JSON 并调用 C# 回调。
// 回调为 nil 时静默丢弃。可从任意 goroutine 安全调用。
func emitEvent(event interface{}) {
	cb := atomic.LoadPointer(&gCallback)
	if cb == nil {
		return
	}
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	cStr := C.CString(string(data))
	C.callCSharp(C.event_callback_t(cb), cStr)
	C.free(unsafe.Pointer(cStr))
}

// ---------- 执行引擎 ----------

// 用于取消正在执行的工作流。
// execMu 保护 execCtx 和 execCancel 的读写，防止多线程竞态。
var (
	execMu     sync.Mutex
	execCtx    context.Context
	execCancel context.CancelFunc
)

//export RobotgoExecute
func RobotgoExecute(workflowPath *C.char, fromStep C.int, debug C.int, inputsJSON *C.char) (ret *C.char) {
	defer func() {
		if r := recover(); r != nil {
			ret = C.CString(fmt.Sprintf(`{"ok":false,"error":"内部错误(panic): %v"}`, r))
		}
	}()

	wfPath := C.GoString(workflowPath)
	from := int(fromStep)

	result := dispatch(func() string {
		// 加载配置
		cfg, err := config.Load(wfPath)
		if err != nil {
			return goErrorJSON("加载工作流失败: " + err.Error())
		}

		// 解析输入变量
		if inputsJSON != nil {
			inputStr := C.GoString(inputsJSON)
			var values map[string]string
			if err := json.Unmarshal([]byte(inputStr), &values); err != nil {
				return goErrorJSON("输入变量解析失败: " + err.Error())
			}
			executor.ResolveInputs(cfg, values)
		}

		// 发送 loaded 事件（模拟当前 serve 协议）
		inputs := make([]InputInfo, len(cfg.Inputs))
		for i, inp := range cfg.Inputs {
			inputs[i] = InputInfo{
				Name:        inp.Name,
				Label:       inp.Label,
				Placeholder: inp.Placeholder,
				Required:    inp.Required,
				Mask:        inp.Mask,
			}
		}
		emitEvent(Event{
			Type:       "loaded",
			OK:         boolPtr(true),
			Name:       cfg.Name,
			TotalSteps: len(cfg.Steps),
			Inputs:     inputs,
		})

		// 创建可取消 context（加锁保护）
		execMu.Lock()
		execCtx, execCancel = context.WithCancel(context.Background())
		execMu.Unlock()
		defer func() {
			execMu.Lock()
			execCancel = nil
			execCtx = nil
			execMu.Unlock()
		}()

		// 创建引擎和执行器
		workDir := filepath.Dir(wfPath)
		eng := engine.NewEngine(workDir, filepath.Join(workDir, "screenshots"))
		defer eng.Close()

		if cfg.Settings.Human.Enabled {
			eng.EnableHuman(true, cfg.Settings.Human.Speed, cfg.Settings.Human.MistakeRate, true, true, false)
		}

		exe := executor.New(eng, cfg, wfPath)
		exe.WithContext(execCtx)
		exe.EnableDebugScreenshots(debug != 0)

		// 设置进度回调 -> emitEvent
		exe.SetCallback(&ffiCallback{})

		// 执行
		var runErr error
		if from > 1 {
			runErr = exe.RunFromStep(from)
		} else {
			runErr = exe.Run()
		}

		if runErr != nil {
			return goErrorJSON(runErr.Error())
		}
		return `{"ok":true}`
	})
	return C.CString(result)
}

//export RobotgoStop
func RobotgoStop() {
	// 在锁内拷贝 execCancel，避免 check-then-act 竞态
	execMu.Lock()
	cancel := execCancel
	execMu.Unlock()
	if cancel != nil {
		cancel()
	}
}

// ---------- executor.ProgressCallback 实现 ----------

// Event 和 InputInfo 复用 serve 包的结构（JSON 标签一致）。
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

type InputInfo struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	Required    bool   `json:"required"`
	Mask        bool   `json:"mask"`
}

func boolPtr(b bool) *bool { return &b }

// ffiCallback 实现 executor.ProgressCallback，将事件推送到 C#。
type ffiCallback struct{}

func (c *ffiCallback) OnWorkflowStart(name string, totalSteps int) {
	emitEvent(Event{Type: "workflow_start", Name: name, TotalSteps: totalSteps})
}
func (c *ffiCallback) OnStepStart(stepIdx, totalSteps int, stepName string, estimatedSec float64) {
	emitEvent(Event{Type: "step_start", Idx: stepIdx, Total: totalSteps, Name: stepName, EstimatedSec: estimatedSec})
}
func (c *ffiCallback) OnActionStart(stepIdx, actionIdx int, actionType, desc string) {
	emitEvent(Event{Type: "action_start", StepIdx: stepIdx, Idx: actionIdx, Action: actionType, Detail: desc})
}
func (c *ffiCallback) OnActionDone(stepIdx, actionIdx int) {
	emitEvent(Event{Type: "action_done", StepIdx: stepIdx, Idx: actionIdx})
}
func (c *ffiCallback) OnStepDone(stepIdx int, err error, screenshotPath string) {
	ev := Event{Type: "step_done", Idx: stepIdx, ScreenshotPath: screenshotPath}
	if err != nil {
		ev.Error = err.Error()
	}
	emitEvent(ev)
}
func (c *ffiCallback) OnWorkflowDone(name string, totalSteps int, err error, totalElapsedSec float64) {
	ok := err == nil
	ev := Event{Type: "workflow_done", OK: &ok, Name: name, TotalSteps: totalSteps, TotalElapsedSec: totalElapsedSec}
	if err != nil {
		ev.Error = err.Error()
	}
	emitEvent(ev)
}
func (c *ffiCallback) OnLog(level executor.LogLevel, message string, stepIdx int) {
	lvlStr := "info"
	switch level {
	case executor.LogWarn:
		lvlStr = "warn"
	case executor.LogError:
		lvlStr = "error"
	}
	emitEvent(Event{Type: "log", Level: lvlStr, Message: message, StepIdx: stepIdx})
}

// ---------- 录制引擎 ----------

// 录制器运行时状态。
// recordMu 保护 recordCmdCh 和 recordRunning 的读写，防止并发竞态和重复启动。
var (
	recordMu      sync.Mutex
	recordCmdCh   chan recorder.RecorderCommand // 接收 C# 命令
	recordRunning bool                          // 防止重复启动
)

//export RobotgoRecordStart
func RobotgoRecordStart(outPath, tplDir *C.char) (ret *C.char) {
	defer func() {
		if r := recover(); r != nil {
			ret = C.CString(fmt.Sprintf(`{"ok":false,"error":"内部错误(panic): %v"}`, r))
		}
	}()

	outputPath := C.GoString(outPath)
	templateDir := C.GoString(tplDir)

	// 加锁保护 recordCmdCh 创建和防重复启动
	recordMu.Lock()
	if recordRunning {
		recordMu.Unlock()
		return C.CString(goErrorJSON("录制器已在运行"))
	}
	recordCmdCh = make(chan recorder.RecorderCommand)
	recordRunning = true
	ch := recordCmdCh
	recordMu.Unlock()

	// 创建 channel 封装的 I/O
	scanner := &channelScanner{ch: ch}
	writer := &eventWriter{} // 将 emitEvent 封装为 io.Writer

	// 在后台 goroutine 启动录制器
	go func() {
		r := recorder.NewPipeIO(outputPath, templateDir, scanner, writer)
		if err := r.Run(); err != nil {
			emitEvent(map[string]interface{}{
				"type":    "record_error",
				"message": err.Error(),
			})
		}
		// 录制结束，重置状态
		recordMu.Lock()
		recordRunning = false
		recordMu.Unlock()
	}()

	return C.CString(`{"ok":true,"status":"started"}`)
}

//export RobotgoRecordCommand
func RobotgoRecordCommand(cmdJSON *C.char) *C.char {
	recordMu.Lock()
	ch := recordCmdCh
	recordMu.Unlock()
	if ch == nil {
		return C.CString(`{"ok":false,"error":"录制器未启动"}`)
	}
	var cmd recorder.RecorderCommand
	if err := json.Unmarshal([]byte(C.GoString(cmdJSON)), &cmd); err != nil {
		return C.CString(fmt.Sprintf(`{"ok":false,"error":"命令解析失败: %v"}`, err))
	}
	select {
	case ch <- cmd:
		return C.CString(`{"ok":true}`)
	default:
		return C.CString(`{"ok":false,"error":"录制器繁忙"}`)
	}
}

//export RobotgoRecordStop
func RobotgoRecordStop() {
	recordMu.Lock()
	ch := recordCmdCh
	recordMu.Unlock()
	if ch != nil {
		select {
		case ch <- recorder.RecorderCommand{Type: "cancel"}:
		default:
		}
	}
}

// channelScanner 将 Go channel 封装为 recorder.scannerInterface.
// 通过实现 Scan/Bytes/Err 方法满足 interface 约束.
type channelScanner struct {
	ch      chan recorder.RecorderCommand
	current recorder.RecorderCommand
	ok      bool
}

func (s *channelScanner) Scan() bool {
	cmd, ok := <-s.ch
	if !ok {
		return false
	}
	s.current = cmd
	s.ok = true
	return true
}

func (s *channelScanner) Bytes() []byte {
	if !s.ok {
		return nil
	}
	data, err := json.Marshal(s.current)
	if err != nil {
		// 序列化失败时返回 nil，由上层录制器忽略该行，避免传播无效数据。
		return nil
	}
	return data
}

func (s *channelScanner) Err() error { return nil }

// eventWriter 将 emitEvent 封装为 io.Writer.
// 录制器的 encoder 写入完整的 JSON 行，直接作为事件发送.
type eventWriter struct{}

func (w *eventWriter) Write(p []byte) (int, error) {
	var ev map[string]interface{}
	if err := json.Unmarshal(p, &ev); err != nil {
		return len(p), nil // 忽略解析失败的行
	}
	emitEvent(ev)
	return len(p), nil
}

// ---------- 截图 ----------

//export RobotgoCapture
func RobotgoCapture(name, outDir *C.char) (ret *C.char) {
	defer func() {
		if r := recover(); r != nil {
			ret = C.CString(fmt.Sprintf(`{"ok":false,"error":"内部错误(panic): %v"}`, r))
		}
	}()

	elemName := C.GoString(name)
	outputDir := C.GoString(outDir)

	result := dispatch(func() string {
		if err := capture.CaptureInteractive(elemName, outputDir); err != nil {
			return goErrorJSON("截图失败: " + err.Error())
		}
		absPath := filepath.Join(outputDir, elemName+".png")
		return fmt.Sprintf(`{"ok":true,"path":%q}`, absPath)
	})
	return C.CString(result)
}
