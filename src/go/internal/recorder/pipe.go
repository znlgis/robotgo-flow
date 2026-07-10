package recorder

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"robotgo-flow/internal/capture"
	"robotgo-flow/internal/config"
)

// scannerInterface 定义 scanner 的行为（Scan / Bytes / Err），
// bufio.Scanner 和测试 mock 均实现此接口，便于在测试中替换为内存实现。
type scannerInterface interface {
	Scan() bool
	Bytes() []byte
	Err() error
}

// PipeRecorder 管道模式录制器，通过 JSON-Line 与 GUI 通信
type PipeRecorder struct {
	outputPath  string
	templateDir string
	scanner     scannerInterface
	encoder     *json.Encoder
	wf          *config.Workflow
}

// NewPipe 创建管道模式录制器
// stdout 用于输出事件，stdin 用于读取命令（分别来自调用方的 os.Stdout / os.Stdin）
func NewPipe(outputPath, templateDir string, stdin, stdout *os.File) *PipeRecorder {
	return &PipeRecorder{
		outputPath:  outputPath,
		templateDir: templateDir,
		scanner:     bufio.NewScanner(stdin),
		encoder:     json.NewEncoder(stdout),
		wf: &config.Workflow{
			Settings: config.Settings{ElementTimeout: 10},
		},
	}
}

// NewPipeIO 创建管道模式录制器，接受自定义 I/O。
// 此变体用于 DLL 嵌入模式——scanner 和 writer 可以是 channel 封装。
func NewPipeIO(outputPath, templateDir string, scanner scannerInterface, writer io.Writer) *PipeRecorder {
	return &PipeRecorder{
		outputPath:  outputPath,
		templateDir: templateDir,
		scanner:     scanner,
		encoder:     json.NewEncoder(writer),
		wf: &config.Workflow{
			Settings: config.Settings{ElementTimeout: 10},
		},
	}
}

func (r *PipeRecorder) sendEvent(ev RecorderEvent) error {
	return r.encoder.Encode(ev)
}

func (r *PipeRecorder) readCommand() (RecorderCommand, error) {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return RecorderCommand{}, fmt.Errorf("读取命令失败: %w", err)
		}
		return RecorderCommand{}, fmt.Errorf("管道已关闭")
	}
	var cmd RecorderCommand
	if err := json.Unmarshal(r.scanner.Bytes(), &cmd); err != nil {
		return RecorderCommand{}, fmt.Errorf("命令解析失败: %w", err)
	}
	return cmd, nil
}

// Run 启动管道模式录制
func (r *PipeRecorder) Run() error {
	// 1. 询问元信息
	if err := r.sendEvent(RecorderEvent{Type: "prompt_meta"}); err != nil {
		return err
	}
	cmd, err := r.readCommand()
	if err != nil {
		return fmt.Errorf("读取元信息失败: %w", err)
	}
	r.wf.Name = strings.TrimSpace(cmd.Name)
	r.wf.Description = strings.TrimSpace(cmd.Description)

	// 2. 确保模板目录存在
	if err := os.MkdirAll(r.templateDir, 0755); err != nil {
		return fmt.Errorf("创建模板目录失败: %w", err)
	}

	// 3. 录制步骤
	stepIdx := 0
	for {
		if err := r.sendEvent(RecorderEvent{Type: "prompt_step_name", StepIndex: stepIdx}); err != nil {
			return err
		}
		cmd, err = r.readCommand()
		if err != nil {
			return fmt.Errorf("读取步骤名失败: %w", err)
		}
		if cmd.Name == "" {
			break
		}
		step := config.Step{Name: strings.TrimSpace(cmd.Name)}

		actionIdx := 0
		for {
			if err := r.sendEvent(RecorderEvent{
				Type: "prompt_action_type", StepIndex: stepIdx, ActionIndex: actionIdx,
			}); err != nil {
				return err
			}
			cmd, err = r.readCommand()
			if err != nil {
				return fmt.Errorf("读取动作类型失败: %w", err)
			}
			actionType := strings.TrimSpace(cmd.ActionType)
			if actionType == "" || actionType == "cancel" {
				break
			}
			action, err := r.handleAction(cmd)
			if err != nil {
				if sendErr := r.sendEvent(RecorderEvent{Type: "record_error", Message: err.Error()}); sendErr != nil {
					return fmt.Errorf("发送错误事件失败（原错误: %w）: %w", err, sendErr)
				}
				return err
			}
			step.Actions = append(step.Actions, *action)
			actionIdx++
		}
		r.wf.Steps = append(r.wf.Steps, step)
		stepIdx++
	}

	// 4. 保存
	if err := r.save(); err != nil {
		if sendErr := r.sendEvent(RecorderEvent{Type: "record_error", Message: err.Error()}); sendErr != nil {
			return fmt.Errorf("发送错误事件失败（原错误: %w）: %w", err, sendErr)
		}
		return err
	}
	if err := r.sendEvent(RecorderEvent{
		Type: "record_done", OutputPath: r.outputPath, StepCount: len(r.wf.Steps),
	}); err != nil {
		return err
	}
	return nil
}

// handleAction 根据命令的动作类型分发给具体处理方法
func (r *PipeRecorder) handleAction(cmd RecorderCommand) (*config.Action, error) {
	switch cmd.ActionType {
	case "click":
		return r.pipeClick(cmd)
	case "double_click":
		return r.pipeDoubleClick(cmd)
	case "right_click":
		return r.pipeRightClick(cmd)
	case "drag":
		return r.pipeDrag(cmd)
	case "type":
		return r.pipeType(cmd)
	case "press_key":
		return r.pipePressKey(cmd)
	case "combo":
		return r.pipeCombo(cmd)
	case "wait_appear":
		return r.pipeWait(cmd)
	case "wait_gone":
		return r.pipeWaitGone(cmd)
	case "scroll":
		return r.pipeScroll(cmd)
	case "open_url":
		return r.pipeOpenURL(cmd)
	case "sleep":
		return r.pipeSleep(cmd)
	case "refresh":
		return &config.Action{Refresh: true}, nil
	case "back":
		return &config.Action{Back: true}, nil
	case "forward":
		return &config.Action{Forward: true}, nil
	default:
		return nil, fmt.Errorf("未知动作类型: %s", cmd.ActionType)
	}
}

// pipeLocate 根据命令中的定位信息解析目标：模板截图或坐标
func (r *PipeRecorder) pipeLocate(cmd RecorderCommand) (any, error) {
	if cmd.TemplatePath != "" {
		absPath := filepath.Join(r.templateDir, cmd.TemplatePath+".png")
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			// 模板不存在，通知 GUI 最小化窗口，然后交互式截图
			if err := r.sendEvent(RecorderEvent{Type: "capture_start", TemplateName: cmd.TemplatePath}); err != nil {
				return nil, fmt.Errorf("发送截图开始事件失败: %w", err)
			}
			ackCmd, err := r.readCommand()
			if err != nil {
				return nil, fmt.Errorf("等待窗口最小化确认失败: %w", err)
			}
			if ackCmd.Type != "window_minimized" {
				return nil, fmt.Errorf("截图过程中收到意外命令: %s", ackCmd.Type)
			}
			if err := capture.CaptureInteractivePipe(cmd.TemplatePath, r.templateDir); err != nil {
				return nil, fmt.Errorf("截图失败: %w", err)
			}
			if err := r.sendEvent(RecorderEvent{Type: "capture_done", TemplatePath: absPath}); err != nil {
				return nil, fmt.Errorf("发送截图完成事件失败: %w", err)
			}
		} else {
			// 模板已存在，复用已有截图
			_ = r.sendEvent(RecorderEvent{Type: "record_info", Message: fmt.Sprintf("模板 %s 已存在，直接复用", cmd.TemplatePath)})
		}
		// 转为相对路径
		rel, err := filepath.Rel(filepath.Dir(r.outputPath), absPath)
		if err != nil {
			return absPath, nil
		}
		return rel, nil
	}
	if cmd.X > 0 || cmd.Y > 0 {
		return map[string]interface{}{"x": cmd.X, "y": cmd.Y}, nil
	}
	return nil, fmt.Errorf("请选择模板截图或输入坐标")
}

// pipeTargetedAction 通用的目标定位动作处理，消除 click/doubleClick/rightClick 的重复代码。
// actionType 决定返回的 Action 字段（"click"、"double_click" 或 "right_click"）。
func (r *PipeRecorder) pipeTargetedAction(actionType string) (*config.Action, error) {
	if err := r.sendEvent(RecorderEvent{Type: "prompt_locate"}); err != nil {
		return nil, fmt.Errorf("发送定位提示失败: %w", err)
	}
	locateCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取定位信息失败: %w", err)
	}
	target, err := r.pipeLocate(locateCmd)
	if err != nil {
		return nil, err
	}
	switch actionType {
	case "click":
		return &config.Action{Click: target}, nil
	case "double_click":
		return &config.Action{DoubleClick: target}, nil
	case "right_click":
		return &config.Action{RightClick: target}, nil
	default:
		return nil, fmt.Errorf("未知的目标动作类型: %s", actionType)
	}
}

func (r *PipeRecorder) pipeClick(cmd RecorderCommand) (*config.Action, error) {
	return r.pipeTargetedAction("click")
}

func (r *PipeRecorder) pipeDoubleClick(cmd RecorderCommand) (*config.Action, error) {
	return r.pipeTargetedAction("double_click")
}

func (r *PipeRecorder) pipeRightClick(cmd RecorderCommand) (*config.Action, error) {
	return r.pipeTargetedAction("right_click")
}

func (r *PipeRecorder) pipeDrag(cmd RecorderCommand) (*config.Action, error) {
	if err := r.sendEvent(RecorderEvent{Type: "prompt_locate", Template: "起点模板"}); err != nil {
		return nil, fmt.Errorf("发送起点定位提示失败: %w", err)
	}
	fromCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取拖拽起点失败: %w", err)
	}
	fromTarget, err := r.pipeLocate(fromCmd)
	if err != nil {
		return nil, fmt.Errorf("起点定位失败: %w", err)
	}
	fromPath, ok := fromTarget.(string)
	if !ok {
		return nil, fmt.Errorf("拖拽起点必须使用模板截图")
	}

	if err := r.sendEvent(RecorderEvent{Type: "prompt_locate", Template: "终点模板"}); err != nil {
		return nil, fmt.Errorf("发送终点定位提示失败: %w", err)
	}
	toCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取拖拽终点失败: %w", err)
	}
	toTarget, err := r.pipeLocate(toCmd)
	if err != nil {
		return nil, fmt.Errorf("终点定位失败: %w", err)
	}
	toPath, ok := toTarget.(string)
	if !ok {
		return nil, fmt.Errorf("拖拽终点必须使用模板截图")
	}

	return &config.Action{Drag: &config.DragSpec{From: fromPath, To: toPath}}, nil
}

func (r *PipeRecorder) pipeType(cmd RecorderCommand) (*config.Action, error) {
	if err := r.sendEvent(RecorderEvent{Type: "prompt_locate", Template: "输入目标"}); err != nil {
		return nil, fmt.Errorf("发送输入目标定位提示失败: %w", err)
	}
	locateCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取输入目标失败: %w", err)
	}
	target, err := r.pipeLocate(locateCmd)
	if err != nil {
		return nil, err
	}
	intoPath, ok := target.(string)
	if !ok {
		return nil, fmt.Errorf("输入目标必须使用模板截图")
	}

	if err := r.sendEvent(RecorderEvent{Type: "prompt_type_text", IsVariableSupported: true}); err != nil {
		return nil, fmt.Errorf("发送文本输入提示失败: %w", err)
	}
	textCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取输入文本失败: %w", err)
	}
	text := strings.TrimSpace(textCmd.Text)
	if textCmd.Variable != nil {
		r.wf.Inputs = append(r.wf.Inputs, config.InputSpec{
			Name:     textCmd.Variable.Name,
			Label:    textCmd.Variable.Label,
			Required: textCmd.Variable.Required,
			Mask:     textCmd.Variable.Mask,
		})
		text = "$input." + textCmd.Variable.Name
	}
	return &config.Action{Type: &config.TypeSpec{Into: intoPath, Text: text}}, nil
}

func (r *PipeRecorder) pipePressKey(cmd RecorderCommand) (*config.Action, error) {
	if err := r.sendEvent(RecorderEvent{Type: "prompt_key"}); err != nil {
		return nil, fmt.Errorf("发送按键提示失败: %w", err)
	}
	keyCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取按键失败: %w", err)
	}
	return &config.Action{Press: strings.TrimSpace(keyCmd.Key)}, nil
}

func (r *PipeRecorder) pipeCombo(cmd RecorderCommand) (*config.Action, error) {
	if err := r.sendEvent(RecorderEvent{Type: "prompt_combo"}); err != nil {
		return nil, fmt.Errorf("发送组合键提示失败: %w", err)
	}
	comboCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取组合键失败: %w", err)
	}
	parts := strings.Split(strings.TrimSpace(comboCmd.Keys), ",")
	keys := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			keys = append(keys, p)
		}
	}
	return &config.Action{Combo: keys}, nil
}

func (r *PipeRecorder) pipeWait(cmd RecorderCommand) (*config.Action, error) {
	if err := r.sendEvent(RecorderEvent{Type: "prompt_locate"}); err != nil {
		return nil, fmt.Errorf("发送定位提示失败: %w", err)
	}
	locateCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取定位信息失败: %w", err)
	}
	target, err := r.pipeLocate(locateCmd)
	if err != nil {
		return nil, err
	}
	return &config.Action{Wait: target}, nil
}

func (r *PipeRecorder) pipeWaitGone(cmd RecorderCommand) (*config.Action, error) {
	if err := r.sendEvent(RecorderEvent{Type: "prompt_locate"}); err != nil {
		return nil, fmt.Errorf("发送定位提示失败: %w", err)
	}
	locateCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取定位信息失败: %w", err)
	}
	target, err := r.pipeLocate(locateCmd)
	if err != nil {
		return nil, err
	}
	path, ok := target.(string)
	if !ok {
		return nil, fmt.Errorf("wait_gone 必须使用模板截图")
	}
	return &config.Action{WaitGone: path}, nil
}

func (r *PipeRecorder) pipeScroll(cmd RecorderCommand) (*config.Action, error) {
	if err := r.sendEvent(RecorderEvent{Type: "prompt_scroll"}); err != nil {
		return nil, fmt.Errorf("发送滚动提示失败: %w", err)
	}
	scrollCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取滚动量失败: %w", err)
	}
	if scrollCmd.Amount == 0 {
		return nil, fmt.Errorf("滚动量不能为零")
	}
	return &config.Action{Scroll: scrollCmd.Amount}, nil
}

func (r *PipeRecorder) pipeOpenURL(cmd RecorderCommand) (*config.Action, error) {
	if err := r.sendEvent(RecorderEvent{Type: "prompt_url"}); err != nil {
		return nil, fmt.Errorf("发送网址提示失败: %w", err)
	}
	urlCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取网址失败: %w", err)
	}
	return &config.Action{OpenURL: strings.TrimSpace(urlCmd.URL)}, nil
}

func (r *PipeRecorder) pipeSleep(cmd RecorderCommand) (*config.Action, error) {
	if err := r.sendEvent(RecorderEvent{Type: "prompt_sleep"}); err != nil {
		return nil, fmt.Errorf("发送延时提示失败: %w", err)
	}
	sleepCmd, err := r.readCommand()
	if err != nil {
		return nil, fmt.Errorf("读取延时失败: %w", err)
	}
	if sleepCmd.Seconds <= 0 {
		return nil, fmt.Errorf("延时秒数必须为正数")
	}
	return &config.Action{Sleep: sleepCmd.Seconds}, nil
}

// save 保存工作流配置到 YAML 文件
func (r *PipeRecorder) save() error {
	return saveWorkflow(r.wf, r.outputPath)
}
