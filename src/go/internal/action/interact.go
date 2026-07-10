package action

import (
	"fmt"

	"robotgo-flow/internal/notify"
)

// PromptAction 弹出输入对话框，获取用户输入后填入目标元素。
type PromptAction struct {
	title   string
	message string
	into    string // 可选：输入前先点击的目标模板
	mask    bool
}

// Execute 弹出 GUI 输入框，可选点击目标元素，然后输入文本。
func (a *PromptAction) Execute(eng Engine) error {
	val := notify.InputBoxStd(a.title, a.message, a.mask)
	if val == "" {
		return fmt.Errorf("弹窗输入: 用户取消或输入为空")
	}

	// 如果指定了目标模板，先定位并点击
	if a.into != "" {
		pt, err := eng.FindElement(a.into)
		if err != nil {
			return fmt.Errorf("弹窗输入: 查找元素 %q 失败: %w", a.into, err)
		}
		if err := eng.Click(pt.X, pt.Y); err != nil {
			return fmt.Errorf("弹窗输入: 点击元素 %q 失败: %w", a.into, err)
		}
	}

	return eng.TypeText(val)
}

// ConfirmAction 弹出确认对话框（是/否）。用户选否则返回错误。
type ConfirmAction struct {
	title   string
	message string
}

// Execute 打印确认提示并读取 y/n 确认。
func (a *ConfirmAction) Execute(eng Engine) error {
	if !notify.ConfirmBoxStd(a.title, a.message) {
		return fmt.Errorf("确认: 用户选择否或取消 (%s)", a.title)
	}
	return nil
}

// NotifyAction 弹出非阻塞通知消息。
type NotifyAction struct {
	title    string
	message  string
	duration float64 // 秒，0 = 手动关闭
}

// Execute 打印通知到 stderr。
func (a *NotifyAction) Execute(eng Engine) error {
	notify.PrintError(a.title, a.message)
	return nil
}
