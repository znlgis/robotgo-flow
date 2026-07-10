package action

import (
	"fmt"

	"robotgo-flow/internal/geom"
)

// targetedAction 提供模板匹配或坐标定位的公共逻辑
type targetedAction struct {
	template string
	coord    *geom.Point
}

func (a *targetedAction) resolveTarget(eng Engine) (int, int, error) {
	if a.coord != nil {
		return a.coord.X, a.coord.Y, nil
	}
	pt, err := eng.FindElement(a.template)
	if err != nil {
		return 0, 0, err
	}
	return pt.X, pt.Y, nil
}

// ClickAction 单击操作
type ClickAction struct {
	targetedAction
}

// Execute 执行单击
func (a *ClickAction) Execute(eng Engine) error {
	x, y, err := a.resolveTarget(eng)
	if err != nil {
		return fmt.Errorf("单击失败: %w", err)
	}
	return eng.Click(x, y)
}

// DoubleClickAction 双击操作
type DoubleClickAction struct {
	targetedAction
}

func (a *DoubleClickAction) Execute(eng Engine) error {
	x, y, err := a.resolveTarget(eng)
	if err != nil {
		return fmt.Errorf("双击失败: %w", err)
	}
	return eng.DoubleClick(x, y)
}

// RightClickAction 右键操作
type RightClickAction struct {
	targetedAction
}

func (a *RightClickAction) Execute(eng Engine) error {
	x, y, err := a.resolveTarget(eng)
	if err != nil {
		return fmt.Errorf("右键单击失败: %w", err)
	}
	return eng.RightClick(x, y)
}
