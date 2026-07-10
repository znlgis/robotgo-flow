package action

import (
	"fmt"

	"robotgo-flow/internal/config"
	"robotgo-flow/internal/geom"
)

// FromConfig 从配置创建动作实例
func FromConfig(cfg config.Action, defaultTimeout int) (Runner, error) {
	switch {
	case cfg.Click != nil:
		return parseClick(cfg.Click)
	case cfg.DoubleClick != nil:
		return parseDoubleClick(cfg.DoubleClick)
	case cfg.RightClick != nil:
		return parseRightClick(cfg.RightClick)
	case cfg.Drag != nil:
		return &DragAction{fromTemplate: cfg.Drag.From, toTemplate: cfg.Drag.To}, nil
	case cfg.Type != nil:
		return &TypeAction{template: cfg.Type.Into, text: cfg.Type.Text}, nil
	case cfg.Press != "":
		return &PressKeyAction{key: cfg.Press}, nil
	case len(cfg.Combo) > 0:
		return &PressComboAction{keys: cfg.Combo}, nil
	case cfg.Wait != nil:
		return parseWait(cfg.Wait, defaultTimeout)
	case cfg.WaitGone != "":
		return &WaitGoneAction{template: cfg.WaitGone, timeout: defaultTimeout}, nil
	case cfg.Scroll != 0:
		return &ScrollAction{amount: cfg.Scroll}, nil
	case cfg.OpenURL != "":
		return &OpenURLAction{url: cfg.OpenURL}, nil
	case cfg.Refresh:
		return &RefreshAction{}, nil
	case cfg.Back:
		return &BackAction{}, nil
	case cfg.Forward:
		return &ForwardAction{}, nil
	case cfg.SwitchTab > 0:
		return &SwitchTabAction{index: cfg.SwitchTab}, nil
	case cfg.Sleep > 0:
		return &SleepAction{seconds: cfg.Sleep}, nil
	case cfg.Prompt != nil:
		return &PromptAction{
			title:   cfg.Prompt.Title,
			message: cfg.Prompt.Message,
			into:    cfg.Prompt.Into,
			mask:    cfg.Prompt.Mask,
		}, nil
	case cfg.Confirm != nil:
		return &ConfirmAction{
			title:   cfg.Confirm.Title,
			message: cfg.Confirm.Message,
		}, nil
	case cfg.Notify != nil:
		return &NotifyAction{
			title:    cfg.Notify.Title,
			message:  cfg.Notify.Message,
			duration: cfg.Notify.Duration,
		}, nil
	default:
		return nil, fmt.Errorf("未知的动作类型")
	}
}

// parseClick 解析 click 字段（string=模板 or map=坐标）
func parseClick(v any) (*ClickAction, error) {
	return parseTargetedAction(v,
		func(tpl string) *ClickAction { return &ClickAction{targetedAction: targetedAction{template: tpl}} },
		func(pt geom.Point) *ClickAction { return &ClickAction{targetedAction: targetedAction{coord: &pt}} },
		"click",
	)
}

func parseDoubleClick(v any) (*DoubleClickAction, error) {
	return parseTargetedAction(v,
		func(tpl string) *DoubleClickAction {
			return &DoubleClickAction{targetedAction: targetedAction{template: tpl}}
		},
		func(pt geom.Point) *DoubleClickAction {
			return &DoubleClickAction{targetedAction: targetedAction{coord: &pt}}
		},
		"double_click",
	)
}

func parseRightClick(v any) (*RightClickAction, error) {
	return parseTargetedAction(v,
		func(tpl string) *RightClickAction {
			return &RightClickAction{targetedAction: targetedAction{template: tpl}}
		},
		func(pt geom.Point) *RightClickAction {
			return &RightClickAction{targetedAction: targetedAction{coord: &pt}}
		},
		"right_click",
	)
}

// parseTargetedAction 是解析 targetedAction 类型操作的通用函数
func parseTargetedAction[T any](v any, makeFromTemplate func(string) T, makeFromCoord func(geom.Point) T, label string) (T, error) {
	switch val := v.(type) {
	case string:
		return makeFromTemplate(val), nil
	case map[string]interface{}:
		x, y, err := parseCoord(val)
		if err != nil {
			var zero T
			return zero, fmt.Errorf("%s 坐标解析失败: %w", label, err)
		}
		return makeFromCoord(geom.Point{X: x, Y: y}), nil
	default:
		var zero T
		return zero, fmt.Errorf("%s: 需要字符串或 {x,y} 格式, 实际为 %T", label, v)
	}
}

// parseWait 解析 wait 字段（string=模板 or map={template,timeout}）
func parseWait(v any, defaultTimeout int) (*WaitAction, error) {
	switch val := v.(type) {
	case string:
		return &WaitAction{template: val, timeout: defaultTimeout}, nil
	case map[string]interface{}:
		tpl, _ := val["template"].(string)
		if tpl == "" {
			return nil, fmt.Errorf("等待: 模板不能为空")
		}
		timeout := defaultTimeout
		if t, ok := toInt(val["timeout"]); ok && t > 0 {
			timeout = t
		}
		return &WaitAction{template: tpl, timeout: timeout}, nil
	default:
		return nil, fmt.Errorf("等待: 需要字符串或 {template,timeout} 格式, 实际为 %T", v)
	}
}

// parseCoord 从 map 中提取坐标
func parseCoord(m map[string]interface{}) (int, int, error) {
	x, okX := toInt(m["x"])
	y, okY := toInt(m["y"])
	if !okX || !okY {
		return 0, 0, fmt.Errorf("坐标需要整型的 x 和 y")
	}
	return x, y, nil
}

// toInt 从 interface{} 提取 int，兼容 yaml.v3 的 int/int64/float64 类型
func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case int:
		return val, true
	case int64:
		return int(val), true
	case float64:
		return int(val), true
	default:
		return 0, false
	}
}
