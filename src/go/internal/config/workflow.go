package config

import (
	"fmt"
)

// Workflow 顶层工作流配置
type Workflow struct {
	Name        string      `yaml:"name"`
	Description string      `yaml:"description,omitempty"`
	Inputs      []InputSpec `yaml:"inputs,omitempty"` // 运行时变量（执行前采集）
	Settings    Settings    `yaml:"settings"`
	Steps       []Step      `yaml:"steps"`
}

// Settings 全局设置
type Settings struct {
	ElementTimeout         int       `yaml:"element_timeout"`
	OnError                string    `yaml:"on_error,omitempty"`                 // abort | skip | retry (default: abort)
	MaxRetries             int       `yaml:"max_retries,omitempty"`              // max retry when on_error=retry (default: 3)
	BrowserRefreshDelay    int       `yaml:"browser_refresh_delay,omitempty"`    // 刷新等待毫秒数 (default: 3000)
	BrowserNavigationDelay int       `yaml:"browser_navigation_delay,omitempty"` // 前进/后退等待毫秒数 (default: 2000)
	BrowserPageLoadDelay   int       `yaml:"browser_page_load_delay,omitempty"`  // 打开URL等待毫秒数 (default: 3000)
	Human                  HumanConf `yaml:"human"`
}

// HumanConf 人类行为模拟配置
type HumanConf struct {
	Enabled     bool    `yaml:"enabled"`
	Speed       float64 `yaml:"speed"`
	MistakeRate float64 `yaml:"mistake_rate"`
}

// Step 一个操作步骤
type Step struct {
	Name    string   `yaml:"name"`
	Actions []Action `yaml:"actions"`
}

// Action 单个动作配置，只有一个字段非零值
type Action struct {
	Click       any       `yaml:"click,omitempty"`
	DoubleClick any       `yaml:"double_click,omitempty"`
	RightClick  any       `yaml:"right_click,omitempty"`
	Drag        *DragSpec `yaml:"drag,omitempty"`
	Type        *TypeSpec `yaml:"type,omitempty"`
	Press       string    `yaml:"press,omitempty"`
	Combo       []string  `yaml:"combo,omitempty"`
	Wait        any       `yaml:"wait,omitempty"`
	WaitGone    string    `yaml:"wait_gone,omitempty"`
	Scroll      int       `yaml:"scroll,omitempty"`
	OpenURL     string    `yaml:"open_url,omitempty"`
	Refresh     bool      `yaml:"refresh,omitempty"`
	Back        bool      `yaml:"back,omitempty"`
	Forward     bool      `yaml:"forward,omitempty"`
	SwitchTab   int       `yaml:"switch_tab,omitempty"`
	Sleep       float64   `yaml:"sleep,omitempty"`

	// 交互动作
	Prompt  *PromptSpec  `yaml:"prompt,omitempty"`
	Confirm *ConfirmSpec `yaml:"confirm,omitempty"`
	Notify  *NotifySpec  `yaml:"notify,omitempty"`
}

// DragSpec 拖拽参数
type DragSpec struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
}

// TypeSpec 输入文本参数
type TypeSpec struct {
	Into string `yaml:"into"`
	Text string `yaml:"text"`
}

// InputSpec 声明工作流执行前需采集的运行变量。
// 录制时生成，执行前通过输入框采集。
type InputSpec struct {
	Name        string `yaml:"name"`                  // 变量名，引用方式为 $input.<name>
	Label       string `yaml:"label"`                 // 输入提示文本
	Required    bool   `yaml:"required"`              // 是否必填
	Mask        bool   `yaml:"mask,omitempty"`        // 是否密码遮罩
	Placeholder string `yaml:"placeholder,omitempty"` // 占位提示文本
}

// PromptSpec 定义执行中弹出的输入对话框。
type PromptSpec struct {
	Title   string `yaml:"title"`
	Message string `yaml:"message"`
	Into    string `yaml:"into,omitempty"` // 可选：输入前先点击的目标模板
	Mask    bool   `yaml:"mask,omitempty"`
}

// ConfirmSpec 定义执行中弹出的确认对话框。
type ConfirmSpec struct {
	Title   string `yaml:"title"`
	Message string `yaml:"message"`
}

// NotifySpec 定义执行中的非阻塞提示通知。
type NotifySpec struct {
	Title    string  `yaml:"title"`
	Message  string  `yaml:"message"`
	Duration float64 `yaml:"duration,omitempty"` // 秒，0 = 手动关闭
}

// ActionType 返回动作的类型标识符（如 "click"、"type" 等），未设置时返回 ""。
// 此方法是动作类型识别的唯一来源，避免多处分发 switch 重复。
func (a *Action) ActionType() string {
	switch {
	case a.Click != nil:
		return "click"
	case a.DoubleClick != nil:
		return "double_click"
	case a.RightClick != nil:
		return "right_click"
	case a.Drag != nil:
		return "drag"
	case a.Type != nil:
		return "type"
	case a.Press != "":
		return "press"
	case len(a.Combo) > 0:
		return "combo"
	case a.Wait != nil:
		return "wait"
	case a.WaitGone != "":
		return "wait_gone"
	case a.Scroll != 0:
		return "scroll"
	case a.OpenURL != "":
		return "open_url"
	case a.Refresh:
		return "refresh"
	case a.Back:
		return "back"
	case a.Forward:
		return "forward"
	case a.SwitchTab > 0:
		return "switch_tab"
	case a.Sleep > 0:
		return "sleep"
	case a.Prompt != nil:
		return "prompt"
	case a.Confirm != nil:
		return "confirm"
	case a.Notify != nil:
		return "notify"
	default:
		return ""
	}
}

// Label 返回动作的中文显示标签，未设置时返回 "未知动作"。
func (a *Action) Label() string {
	switch {
	case a.Click != nil:
		return fmt.Sprintf("单击 (%v)", a.Click)
	case a.DoubleClick != nil:
		return fmt.Sprintf("双击 (%v)", a.DoubleClick)
	case a.RightClick != nil:
		return fmt.Sprintf("右键 (%v)", a.RightClick)
	case a.Drag != nil:
		return fmt.Sprintf("拖拽 %s→%s", a.Drag.From, a.Drag.To)
	case a.Type != nil:
		return fmt.Sprintf("输入: %s", a.Type.Text)
	case a.Press != "":
		return fmt.Sprintf("按键: %s", a.Press)
	case len(a.Combo) > 0:
		return fmt.Sprintf("组合键: %v", a.Combo)
	case a.Wait != nil:
		return fmt.Sprintf("等待模板: %v", a.Wait)
	case a.WaitGone != "":
		return fmt.Sprintf("等待消失: %s", a.WaitGone)
	case a.Scroll != 0:
		return fmt.Sprintf("滚动: %d", a.Scroll)
	case a.OpenURL != "":
		return fmt.Sprintf("打开URL: %s", a.OpenURL)
	case a.Refresh:
		return "刷新页面"
	case a.Back:
		return "后退"
	case a.Forward:
		return "前进"
	case a.SwitchTab > 0:
		return fmt.Sprintf("切换标签: %d", a.SwitchTab)
	case a.Sleep > 0:
		return fmt.Sprintf("等待 %.1fs", a.Sleep)
	case a.Prompt != nil:
		return fmt.Sprintf("弹窗输入: %s", a.Prompt.Title)
	case a.Confirm != nil:
		return fmt.Sprintf("确认: %s", a.Confirm.Title)
	case a.Notify != nil:
		return fmt.Sprintf("提示: %s", a.Notify.Title)
	default:
		return "未知动作"
	}
}
