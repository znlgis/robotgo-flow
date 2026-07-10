package recorder

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"robotgo-flow/internal/config"
)

// --- 测试辅助 ---

// mockScanner 实现 scannerInterface，允许在测试中预置输入行
type mockScanner struct {
	lines [][]byte
	idx   int
}

func (s *mockScanner) Scan() bool {
	if s.idx >= len(s.lines) {
		return false
	}
	s.idx++
	return true
}

func (s *mockScanner) Bytes() []byte {
	return s.lines[s.idx-1]
}

func (s *mockScanner) Err() error {
	return nil
}

// newPipeForTest 创建 PipeRecorder 的可测试实例，
// scanner 使用 mockScanner，encoder 输出到 io.Discard。
func newPipeForTest(outputPath, templateDir string, jsonLines []string) *PipeRecorder {
	lines := make([][]byte, len(jsonLines))
	for i, l := range jsonLines {
		lines[i] = []byte(l)
	}
	return &PipeRecorder{
		outputPath:  outputPath,
		templateDir: templateDir,
		scanner:     &mockScanner{lines: lines},
		encoder:     json.NewEncoder(io.Discard),
		wf: &config.Workflow{
			Settings: config.Settings{ElementTimeout: 10},
		},
	}
}

// mustMarshal 将值序列化为 JSON 字符串，序列化失败时触发 t.Fatal。
func mustMarshal(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("序列化 JSON 失败: %v", err)
	}
	return string(b)
}

// --- readCommand 测试 ---

func TestPipeRecorder_ReadCommand_Success(t *testing.T) {
	tmpDir := t.TempDir()
	r := newPipeForTest(
		filepath.Join(tmpDir, "out.yaml"),
		filepath.Join(tmpDir, "tpl"),
		[]string{mustMarshal(t, RecorderCommand{Type: "meta", Name: "测试流程"})},
	)

	cmd, err := r.readCommand()
	if err != nil {
		t.Fatalf("readCommand 失败: %v", err)
	}
	if cmd.Type != "meta" {
		t.Errorf("期望 Type='meta', 实际='%s'", cmd.Type)
	}
	if cmd.Name != "测试流程" {
		t.Errorf("期望 Name='测试流程', 实际='%s'", cmd.Name)
	}
}

func TestPipeRecorder_ReadCommand_EmptyObject(t *testing.T) {
	tmpDir := t.TempDir()
	r := newPipeForTest(
		filepath.Join(tmpDir, "out.yaml"),
		filepath.Join(tmpDir, "tpl"),
		[]string{"{}"},
	)

	cmd, err := r.readCommand()
	if err != nil {
		t.Fatalf("空对象 JSON 应成功解析, 但返回错误: %v", err)
	}
	// 空对象的字段均为零值
	if cmd.Type != "" || cmd.Name != "" || cmd.ActionType != "" {
		t.Errorf("空对象命令字段应为零值, 实际: Type=%q Name=%q ActionType=%q",
			cmd.Type, cmd.Name, cmd.ActionType)
	}
}

func TestPipeRecorder_ReadCommand_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	r := newPipeForTest(
		filepath.Join(tmpDir, "out.yaml"),
		filepath.Join(tmpDir, "tpl"),
		[]string{"{bad json"},
	)

	_, err := r.readCommand()
	if err == nil {
		t.Fatal("格式错误的 JSON 应返回错误")
	}
	if !strings.Contains(err.Error(), "命令解析失败") {
		t.Errorf("错误消息应包含'命令解析失败', 实际: %v", err)
	}
}

func TestPipeRecorder_ReadCommand_EOF(t *testing.T) {
	tmpDir := t.TempDir()
	r := newPipeForTest(
		filepath.Join(tmpDir, "out.yaml"),
		filepath.Join(tmpDir, "tpl"),
		nil, // 空行列表，Scanner.Scan() 立即返回 false
	)

	_, err := r.readCommand()
	if err == nil {
		t.Fatal("EOF 时应返回错误")
	}
	if !strings.Contains(err.Error(), "管道已关闭") {
		t.Errorf("错误消息应包含'管道已关闭', 实际: %v", err)
	}
}

// --- handleAction 测试 ---

// TestPipeRecorder_HandleAction_NonInteractive 测试无需额外 stdin 交互的动作类型
func TestPipeRecorder_HandleAction_NonInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	r := newPipeForTest(
		filepath.Join(tmpDir, "out.yaml"),
		filepath.Join(tmpDir, "tpl"),
		nil,
	)

	tests := []struct {
		name       string
		cmd        RecorderCommand
		checkField func(t *testing.T, act *config.Action)
	}{
		{
			name: "refresh",
			cmd:  RecorderCommand{ActionType: "refresh"},
			checkField: func(t *testing.T, act *config.Action) {
				if !act.Refresh {
					t.Error("Refresh 应为 true")
				}
			},
		},
		{
			name: "back",
			cmd:  RecorderCommand{ActionType: "back"},
			checkField: func(t *testing.T, act *config.Action) {
				if !act.Back {
					t.Error("Back 应为 true")
				}
			},
		},
		{
			name: "forward",
			cmd:  RecorderCommand{ActionType: "forward"},
			checkField: func(t *testing.T, act *config.Action) {
				if !act.Forward {
					t.Error("Forward 应为 true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			act, err := r.handleAction(tt.cmd)
			if err != nil {
				t.Fatalf("handleAction(%s) 失败: %v", tt.name, err)
			}
			if act == nil {
				t.Fatalf("handleAction(%s) 返回 nil action", tt.name)
			}
			tt.checkField(t, act)
		})
	}
}

// TestPipeRecorder_HandleAction_Interactive 测试需要额外 stdin 交互的动作类型
func TestPipeRecorder_HandleAction_Interactive(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		cmd          RecorderCommand
		scannerLines []string
		expectError  bool // 若为 true，则期望 handleAction 返回错误
		checkField   func(t *testing.T, act *config.Action)
	}{
		{
			name:         "click with coords",
			cmd:          RecorderCommand{ActionType: "click"},
			scannerLines: []string{`{"x":100,"y":200}`},
			checkField: func(t *testing.T, act *config.Action) {
				m, ok := act.Click.(map[string]interface{})
				if !ok {
					t.Fatal("Click 字段类型应为 map[string]interface{}")
				}
				if fmt.Sprint(m["x"]) != "100" || fmt.Sprint(m["y"]) != "200" {
					t.Errorf("期望坐标 (100,200), 实际 (%v,%v)", m["x"], m["y"])
				}
			},
		},
		{
			name:         "double_click with coords",
			cmd:          RecorderCommand{ActionType: "double_click"},
			scannerLines: []string{`{"x":50,"y":75}`},
			checkField: func(t *testing.T, act *config.Action) {
				if act.DoubleClick == nil {
					t.Fatal("DoubleClick 字段为 nil")
				}
			},
		},
		{
			name:         "right_click with coords",
			cmd:          RecorderCommand{ActionType: "right_click"},
			scannerLines: []string{`{"x":10,"y":20}`},
			checkField: func(t *testing.T, act *config.Action) {
				if act.RightClick == nil {
					t.Fatal("RightClick 字段为 nil")
				}
			},
		},
		{
			name:         "click_without_coords_or_template",
			cmd:          RecorderCommand{ActionType: "click"},
			scannerLines: []string{`{}`},
			expectError:  true,
		},
		{
			name:         "press_key",
			cmd:          RecorderCommand{ActionType: "press_key"},
			scannerLines: []string{`{"type":"key_name","key":"enter"}`},
			checkField: func(t *testing.T, act *config.Action) {
				if act.Press != "enter" {
					t.Errorf("期望 Press='enter', 实际='%s'", act.Press)
				}
			},
		},
		{
			name:         "combo",
			cmd:          RecorderCommand{ActionType: "combo"},
			scannerLines: []string{`{"type":"combo_keys","keys":"ctrl,c"}`},
			checkField: func(t *testing.T, act *config.Action) {
				if len(act.Combo) != 2 || act.Combo[0] != "ctrl" || act.Combo[1] != "c" {
					t.Errorf("期望 Combo=['ctrl','c'], 实际=%v", act.Combo)
				}
			},
		},
		{
			name:         "open_url",
			cmd:          RecorderCommand{ActionType: "open_url"},
			scannerLines: []string{`{"type":"url","url":"https://example.com"}`},
			checkField: func(t *testing.T, act *config.Action) {
				if act.OpenURL != "https://example.com" {
					t.Errorf("期望 OpenURL='https://example.com', 实际='%s'", act.OpenURL)
				}
			},
		},
		{
			name:         "sleep",
			cmd:          RecorderCommand{ActionType: "sleep"},
			scannerLines: []string{`{"type":"sleep_seconds","seconds":2.5}`},
			checkField: func(t *testing.T, act *config.Action) {
				if act.Sleep != 2.5 {
					t.Errorf("期望 Sleep=2.5, 实际=%v", act.Sleep)
				}
			},
		},
		{
			name:         "scroll",
			cmd:          RecorderCommand{ActionType: "scroll"},
			scannerLines: []string{`{"type":"scroll_amount","amount":-100}`},
			checkField: func(t *testing.T, act *config.Action) {
				if act.Scroll != -100 {
					t.Errorf("期望 Scroll=-100, 实际=%d", act.Scroll)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newPipeForTest(
				filepath.Join(tmpDir, "out.yaml"),
				filepath.Join(tmpDir, "tpl"),
				tt.scannerLines,
			)
			act, err := r.handleAction(tt.cmd)
			if tt.expectError {
				if err == nil {
					t.Error("缺少坐标和模板路径时应返回错误")
				}
				return
			}
			if err != nil {
				t.Fatalf("handleAction(%s) 失败: %v", tt.name, err)
			}
			if act == nil {
				t.Fatalf("handleAction(%s) 返回 nil action", tt.name)
			}
			tt.checkField(t, act)
		})
	}
}

// TestPipeRecorder_HandleAction_UnknownType 测试未知动作类型
func TestPipeRecorder_HandleAction_UnknownType(t *testing.T) {
	tmpDir := t.TempDir()
	r := newPipeForTest(
		filepath.Join(tmpDir, "out.yaml"),
		filepath.Join(tmpDir, "tpl"),
		nil,
	)

	_, err := r.handleAction(RecorderCommand{ActionType: "nonexistent"})
	if err == nil {
		t.Fatal("未知动作类型应返回错误")
	}
	if !strings.Contains(err.Error(), "未知动作类型") {
		t.Errorf("错误消息应包含'未知动作类型', 实际: %v", err)
	}
}

// TestPipeRecorder_HandleAction_SleepNegative 测试负秒数边界
func TestPipeRecorder_HandleAction_SleepNegative(t *testing.T) {
	tmpDir := t.TempDir()
	r := newPipeForTest(
		filepath.Join(tmpDir, "out.yaml"),
		filepath.Join(tmpDir, "tpl"),
		[]string{`{"type":"sleep_seconds","seconds":-1}`},
	)

	_, err := r.handleAction(RecorderCommand{ActionType: "sleep"})
	if err == nil {
		t.Fatal("负秒数应返回错误")
	}
	if !strings.Contains(err.Error(), "必须为正数") {
		t.Errorf("错误消息应包含'必须为正数', 实际: %v", err)
	}
}

// --- save 测试 ---

func TestPipeRecorder_Save(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "workflow.yaml")
	r := newPipeForTest(outputPath, filepath.Join(tmpDir, "tpl"), nil)

	r.wf.Name = "测试工作流"
	r.wf.Description = "用于测试的保存功能"
	r.wf.Steps = []config.Step{
		{
			Name: "步骤1",
			Actions: []config.Action{
				{Click: map[string]interface{}{"x": 100, "y": 200}},
				{Press: "enter"},
			},
		},
		{
			Name: "步骤2",
			Actions: []config.Action{
				{Sleep: 1.5},
				{Refresh: true},
			},
		},
	}

	if err := r.save(); err != nil {
		t.Fatalf("save 失败: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("读取输出文件失败: %v", err)
	}
	content := string(data)

	checks := []string{
		"测试工作流",
		"用于测试的保存功能",
		"步骤1",
		"步骤2",
		"click:",
		"press: enter",
		"sleep: 1.5",
		"refresh: true",
	}
	for _, c := range checks {
		if !strings.Contains(content, c) {
			t.Errorf("输出 YAML 应包含 %q", c)
		}
	}
}

// --- 变量采集测试 ---

func TestPipeRecorder_VariableCollection(t *testing.T) {
	tmpDir := t.TempDir()
	r := newPipeForTest(
		filepath.Join(tmpDir, "out.yaml"),
		filepath.Join(tmpDir, "tpl"),
		nil,
	)

	// 模拟 pipeType 追加 InputSpec 的逻辑
	r.wf.Inputs = append(r.wf.Inputs, config.InputSpec{
		Name:     "username",
		Label:    "用户名",
		Required: true,
		Mask:     false,
	})
	r.wf.Inputs = append(r.wf.Inputs, config.InputSpec{
		Name:     "password",
		Label:    "密码",
		Required: true,
		Mask:     true,
	})

	if len(r.wf.Inputs) != 2 {
		t.Fatalf("期望 2 个 Input, 实际 %d", len(r.wf.Inputs))
	}

	input1 := r.wf.Inputs[0]
	if input1.Name != "username" {
		t.Errorf("第一个 Input 名称期望 'username', 实际 '%s'", input1.Name)
	}
	if input1.Label != "用户名" {
		t.Errorf("第一个 Input 标签期望 '用户名', 实际 '%s'", input1.Label)
	}
	if !input1.Required {
		t.Error("第一个 Input 应为必填")
	}
	if input1.Mask {
		t.Error("第一个 Input 不应为密码遮罩")
	}

	input2 := r.wf.Inputs[1]
	if input2.Name != "password" {
		t.Errorf("第二个 Input 名称期望 'password', 实际 '%s'", input2.Name)
	}
	if !input2.Mask {
		t.Error("第二个 Input 应为密码遮罩")
	}
}

// --- sendEvent 与事件响应循环测试 ---

func TestPipeRecorder_SendEvent(t *testing.T) {
	tmpDir := t.TempDir()
	// 用 bytes.Buffer 捕获 stdout 输出，验证 sendEvent 写入的内容
	var stdout bytes.Buffer
	r := &PipeRecorder{
		outputPath:  filepath.Join(tmpDir, "out.yaml"),
		templateDir: filepath.Join(tmpDir, "tpl"),
		scanner:     &mockScanner{},
		encoder:     json.NewEncoder(&stdout),
		wf: &config.Workflow{
			Settings: config.Settings{ElementTimeout: 10},
		},
	}

	ev := RecorderEvent{Type: "prompt_meta"}
	if err := r.sendEvent(ev); err != nil {
		t.Fatalf("sendEvent 失败: %v", err)
	}

	// 输出应包含完整的 JSON 行
	line := strings.TrimSpace(stdout.String())
	var decoded RecorderEvent
	if err := json.Unmarshal([]byte(line), &decoded); err != nil {
		t.Fatalf("输出不是有效的 JSON: %v (内容: %q)", err, line)
	}
	if decoded.Type != "prompt_meta" {
		t.Errorf("期望 Type='prompt_meta', 实际='%s'", decoded.Type)
	}
}

// --- Run 核心流程测试 ---

// TestPipeRecorder_Run_FullCycle 模拟完整的录制流程：
// meta → 一个步骤含两个动作 → 空步骤名退出 → 保存
func TestPipeRecorder_Run_FullCycle(t *testing.T) {
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "full_cycle.yaml")

	commands := []RecorderCommand{
		// 1. meta
		{Type: "meta", Name: "完整循环测试"},
		// 2. step 1 name
		{Name: "登录步骤"},
		// 3. action 1: click
		{ActionType: "click"},
		// 3b. 定位响应（click 的 prompt_locate 读取此命令）
		{X: 300, Y: 400},
		// 4. action 2: refresh
		{ActionType: "refresh"},
		// 5. end this step's actions (empty action_type)
		{ActionType: ""},
		// 6. step 2 name (empty to end recording)
		{Name: ""},
	}

	jsonLines := make([]string, len(commands))
	for i, cmd := range commands {
		jsonLines[i] = mustMarshal(t, cmd)
	}

	// 用 bytes.Buffer 捕获事件输出，便于验证
	var stdout bytes.Buffer
	r := &PipeRecorder{
		outputPath:  outputPath,
		templateDir: filepath.Join(tmpDir, "tpl"),
		scanner:     &mockScanner{lines: makeBytesLines(jsonLines)},
		encoder:     json.NewEncoder(&stdout),
		wf: &config.Workflow{
			Settings: config.Settings{ElementTimeout: 10},
		},
	}

	if err := r.Run(); err != nil {
		t.Fatalf("Run 失败: %v", err)
	}

	// 验证事件输出
	eventOutput := stdout.String()
	for _, expectedType := range []string{
		"prompt_meta",
		"prompt_step_name",
		"prompt_action_type",
		"prompt_locate",
		"record_done",
	} {
		if !strings.Contains(eventOutput, fmt.Sprintf(`"type":"%s"`, expectedType)) {
			t.Errorf("事件输出应包含 %q", expectedType)
		}
	}
	if !strings.Contains(eventOutput, `"output_path":"`) {
		t.Error("record_done 事件应包含 output_path")
	}

	// 验证保存的 YAML
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("读取输出文件失败: %v", err)
	}
	content := string(data)

	required := []string{
		"完整循环测试",
		"登录步骤",
		"click:",
		"refresh: true",
	}
	for _, s := range required {
		if !strings.Contains(content, s) {
			t.Errorf("输出 YAML 应包含 %q", s)
		}
	}

	// 验证工作流结构
	if r.wf.Name != "完整循环测试" {
		t.Errorf("工作流名称期望 '完整循环测试', 实际 '%s'", r.wf.Name)
	}
	if len(r.wf.Steps) != 1 {
		t.Errorf("期望 1 个步骤, 实际 %d", len(r.wf.Steps))
	}
	if len(r.wf.Steps[0].Actions) != 2 {
		t.Errorf("期望 2 个动作, 实际 %d", len(r.wf.Steps[0].Actions))
	}
}

// makeBytesLines converts string slice to [][]byte for mockScanner
func makeBytesLines(lines []string) [][]byte {
	out := make([][]byte, len(lines))
	for i, l := range lines {
		out[i] = []byte(l)
	}
	return out
}

// --- 边界测试 ---

// TestPipeRecorder_Run_MetaReadError 测试 meta 读取失败（EOF）
func TestPipeRecorder_Run_MetaReadError(t *testing.T) {
	tmpDir := t.TempDir()
	r := newPipeForTest(
		filepath.Join(tmpDir, "out.yaml"),
		filepath.Join(tmpDir, "tpl"),
		nil, // 空输入，meta 读取立即 EOF
	)

	err := r.Run()
	if err == nil {
		t.Fatal("meta 读取失败时应返回错误")
	}
	if !strings.Contains(err.Error(), "读取元信息失败") {
		t.Errorf("错误消息应包含'读取元信息失败', 实际: %v", err)
	}
}

// TestPipeRecorder_Run_StepNameReadError 测试步骤名读取失败
func TestPipeRecorder_Run_StepNameReadError(t *testing.T) {
	tmpDir := t.TempDir()
	// 先发送 prompt_meta，再发送 meta 命令 → 然后发 prompt_step_name，但读不到步骤名
	r := newPipeForTest(
		filepath.Join(tmpDir, "out.yaml"),
		filepath.Join(tmpDir, "tpl"),
		[]string{mustMarshal(t, RecorderCommand{Type: "meta", Name: "测试"})},
	)

	err := r.Run()
	if err == nil {
		t.Fatal("步骤名读取失败时应返回错误")
	}
	if !strings.Contains(err.Error(), "读取步骤名失败") {
		t.Errorf("错误消息应包含'读取步骤名失败', 实际: %v", err)
	}
}

// TestPipeRecorder_Run_ActionTypeReadError 测试动作类型读取失败
func TestPipeRecorder_Run_ActionTypeReadError(t *testing.T) {
	tmpDir := t.TempDir()
	r := newPipeForTest(
		filepath.Join(tmpDir, "out.yaml"),
		filepath.Join(tmpDir, "tpl"),
		[]string{
			mustMarshal(t, RecorderCommand{Type: "meta", Name: "测试"}),
			mustMarshal(t, RecorderCommand{Name: "步骤1"}), // 步骤名
			// 没有后续命令，动作类型读取失败
		},
	)

	err := r.Run()
	if err == nil {
		t.Fatal("动作类型读取失败时应返回错误")
	}
	if !strings.Contains(err.Error(), "读取动作类型失败") {
		t.Errorf("错误消息应包含'读取动作类型失败', 实际: %v", err)
	}
}

// TestPipeRecorder_Scanner_ImplementsInterface 编译期验证 bufio.Scanner 实现接口
func TestPipeRecorder_Scanner_ImplementsInterface(t *testing.T) {
	// 此测试在编译期即验证，运行时不做事
	// 若 *bufio.Scanner 未实现 scannerInterface，go build 即报错
}
