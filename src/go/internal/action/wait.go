package action

import "fmt"

// WaitAction 等待元素出现
type WaitAction struct {
	template string
	timeout  int
}

func (a *WaitAction) Execute(eng Engine) error {
	if err := eng.WaitForElement(a.template, a.timeout); err != nil {
		return fmt.Errorf("等待失败: %w", err)
	}
	return nil
}

// WaitGoneAction 等待元素消失
type WaitGoneAction struct {
	template string
	timeout  int
}

func (a *WaitGoneAction) Execute(eng Engine) error {
	if err := eng.WaitForElementGone(a.template, a.timeout); err != nil {
		return fmt.Errorf("等待消失失败: %w", err)
	}
	return nil
}
