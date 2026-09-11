// Package protocol 定义 Go 引擎与 GUI 之间共享的 JSON-Line 协议消息类型。
// serve（子进程模式）与 ffi（DLL 模式）共用同一套结构定义，避免两侧字段漂移。
package protocol

import "robotgo-flow/internal/config"

// Event 表示发给 GUI 的单条事件对象（每行一个 JSON 对象）。
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

// InputInfo 描述一个运行时变量的元数据（随 loaded 事件下发）。
type InputInfo struct {
	Name        string `json:"name"`
	Label       string `json:"label"`
	Placeholder string `json:"placeholder,omitempty"`
	Required    bool   `json:"required"`
	Mask        bool   `json:"mask"`
}

// Command 表示从 GUI 接收的命令。
type Command struct {
	Type   string            `json:"type"`
	Values map[string]string `json:"values,omitempty"`
}

// InputInfos 将 config.InputSpec 切片转换为协议 InputInfo 切片。
func InputInfos(inputs []config.InputSpec) []InputInfo {
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

// BoolPtr 返回指向 b 的指针，便于表达 "ok" 字段的三态（缺省/true/false）。
func BoolPtr(b bool) *bool { return &b }
