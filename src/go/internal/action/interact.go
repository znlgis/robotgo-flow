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

// Execute 请求用户输入，可选点击目标元素，然后输入文本。
// 取值通过 notify.Interactor 完成：CLI 下读取 stdin，serve/DLL 等宿主可替换实现。
func (a *PromptAction) Execute(eng Engine) error {
	val, err := notify.InputBoxStd(a.title, a.message, a.mask)
	if err != nil {
		return fmt.Errorf("弹窗输入: %w", err)
	}
	if val == "" {
		return fmt.Errorf("弹窗输入: 用户取消或输入为空 (%s)", a.title)
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

// Execute 请求用户确认；用户选择否或无法交互时返回错误（由 on_error 策略决定后续处理）。
func (a *ConfirmAction) Execute(eng Engine) error {
	ok, err := notify.ConfirmBoxStd(a.title, a.message)
	if err != nil {
		return fmt.Errorf("确认: %w (%s)", err, a.title)
	}
	if !ok {
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

// Execute 输出通知信息到 stderr（不阻塞执行流程）。
func (a *NotifyAction) Execute(eng Engine) error {
	notify.PrintError(a.title, a.message)
	return nil
}
