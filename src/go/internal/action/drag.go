package action

import "fmt"

// DragAction 拖拽操作
type DragAction struct {
	fromTemplate string
	toTemplate   string
}

func (a *DragAction) Execute(eng Engine) error {
	from, err := eng.FindElement(a.fromTemplate)
	if err != nil {
		return fmt.Errorf("拖拽起点失败: %w", err)
	}
	to, err := eng.FindElement(a.toTemplate)
	if err != nil {
		return fmt.Errorf("拖拽终点失败: %w", err)
	}
	return eng.Drag(from.X, from.Y, to.X, to.Y)
}
