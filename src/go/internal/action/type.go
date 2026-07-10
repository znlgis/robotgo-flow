package action

import "fmt"

// TypeAction 输入文本操作
type TypeAction struct {
	template string
	text     string
}

func (a *TypeAction) Execute(eng Engine) error {
	pt, err := eng.FindElement(a.template)
	if err != nil {
		return fmt.Errorf("输入操作: 查找输入框失败: %w", err)
	}
	if err := eng.Click(pt.X, pt.Y); err != nil {
		return fmt.Errorf("输入操作: 点击输入框失败: %w", err)
	}
	return eng.TypeText(a.text)
}
