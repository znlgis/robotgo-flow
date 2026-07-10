package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"robotgo-flow/internal/encoding"
)

// Load 从 YAML 文件加载工作流配置，自动将 GBK 编码转为 UTF-8
func Load(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("加载配置失败: %w", err)
	}

	// 如果 YAML 文件是 GBK 编码（Windows 下常见），转为 UTF-8
	if utf8Data, ok := encoding.ToUTF8(data); ok {
		data = utf8Data
	}

	var wf Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	// 设置默认值（必须在 Validate 之前，以确保默认超时等字段非零）
	wf.applyDefaults()

	if err := wf.Validate(filepath.Dir(path)); err != nil {
		return nil, fmt.Errorf("校验配置失败: %w", err)
	}

	return &wf, nil
}

// Validate 校验配置完整性
// workflowDir 用于将相对模板路径解析为绝对路径（通常为 YAML 文件所在目录）
func (w *Workflow) Validate(workflowDir string) error {
	if w.Name == "" {
		return fmt.Errorf("工作流名称不能为空")
	}
	if len(w.Steps) == 0 {
		return fmt.Errorf("至少需要一个步骤")
	}
	for i, step := range w.Steps {
		if step.Name == "" {
			return fmt.Errorf("步骤 %d: 名称不能为空", i+1)
		}
		if len(step.Actions) == 0 {
			return fmt.Errorf("步骤 %q: 至少需要一个动作", step.Name)
		}
		for j, action := range step.Actions {
			if action.isEmpty() {
				return fmt.Errorf("步骤 %q 动作 %d: 未指定动作类型", step.Name, j+1)
			}
			if err := action.validateSingleAction(); err != nil {
				return fmt.Errorf("步骤 %q 动作 %d: %w", step.Name, j+1, err)
			}
			// 校验引用的模板文件存在（相对路径以 workflowDir 为基准解析）
			if err := action.validateTemplates(workflowDir); err != nil {
				return fmt.Errorf("步骤 %q 动作 %d: %w", step.Name, j+1, err)
			}
			// 校验 map 类型字段完整性
			if err := action.validateMapFields(); err != nil {
				return fmt.Errorf("步骤 %q 动作 %d: %w", step.Name, j+1, err)
			}
			// 校验交互动作字段
			if action.Prompt != nil {
				if action.Prompt.Title == "" {
					return fmt.Errorf("步骤 %q 动作 %d: 弹窗标题不能为空", step.Name, j+1)
				}
				if action.Prompt.Message == "" {
					return fmt.Errorf("步骤 %q 动作 %d: 弹窗消息不能为空", step.Name, j+1)
				}
			}
			if action.Confirm != nil {
				if action.Confirm.Title == "" {
					return fmt.Errorf("步骤 %q 动作 %d: 确认标题不能为空", step.Name, j+1)
				}
				if action.Confirm.Message == "" {
					return fmt.Errorf("步骤 %q 动作 %d: 确认消息不能为空", step.Name, j+1)
				}
			}
		}
	}
	if w.Settings.ElementTimeout <= 0 {
		return fmt.Errorf("元素超时时间必须为正数")
	}
	if w.Settings.Human.Enabled {
		s := w.Settings.Human.Speed
		if s < 0.1 || s > 5.0 {
			return fmt.Errorf("human.speed 必须为 0.1 ~ 5.0，当前值为 %.2f", s)
		}
		m := w.Settings.Human.MistakeRate
		if m < 0.0 || m > 1.0 {
			return fmt.Errorf("human.mistake_rate 必须为 0.0 ~ 1.0，当前值为 %.2f", m)
		}
	}
	return nil
}

// isEmpty 检查 Action 是否所有字段都为零值
func (a *Action) isEmpty() bool {
	return a.ActionType() == ""
}

// validateMapFields 校验 map 类型 Action 字段完整性
func (a *Action) validateMapFields() error {
	if a.Drag != nil {
		if a.Drag.From == "" {
			return fmt.Errorf("拖拽起点不能为空")
		}
		if a.Drag.To == "" {
			return fmt.Errorf("拖拽终点不能为空")
		}
	}
	if a.Type != nil {
		if a.Type.Into == "" {
			return fmt.Errorf("输入目标不能为空")
		}
		if a.Type.Text == "" {
			return fmt.Errorf("输入文本不能为空")
		}
	}
	return nil
}

// validateSingleAction 校验 Action 只有一个字段被设置
func (a *Action) validateSingleAction() error {
	count := 0
	if a.Click != nil {
		count++
	}
	if a.DoubleClick != nil {
		count++
	}
	if a.RightClick != nil {
		count++
	}
	if a.Drag != nil {
		count++
	}
	if a.Type != nil {
		count++
	}
	if a.Press != "" {
		count++
	}
	if len(a.Combo) > 0 {
		count++
	}
	if a.Wait != nil {
		count++
	}
	if a.WaitGone != "" {
		count++
	}
	if a.Scroll != 0 {
		count++
	}
	if a.OpenURL != "" {
		count++
	}
	if a.Refresh {
		count++
	}
	if a.Back {
		count++
	}
	if a.Forward {
		count++
	}
	if a.SwitchTab > 0 {
		count++
	}
	if a.Sleep > 0 {
		count++
	}
	if a.Prompt != nil {
		count++
	}
	if a.Confirm != nil {
		count++
	}
	if a.Notify != nil {
		count++
	}
	if count > 1 {
		return fmt.Errorf("每个 action 只能包含一个操作类型，当前包含 %d 个", count)
	}
	return nil
}

// validateTemplates 校验 Action 中引用的模板文件存在
// baseDir 用于将相对路径解析为绝对路径
func (a *Action) validateTemplates(baseDir string) error {
	templates := a.templatePaths()
	for _, tpl := range templates {
		if tpl == "" {
			return fmt.Errorf("模板路径不能为空")
		}
		resolved := tpl
		if !filepath.IsAbs(tpl) {
			resolved = filepath.Join(baseDir, tpl)
		}
		if _, err := os.Stat(resolved); os.IsNotExist(err) {
			return fmt.Errorf("模板未找到: %s", tpl)
		}
	}
	return nil
}

// templatePaths 返回 Action 中引用的所有模板文件路径
func (a *Action) templatePaths() []string {
	var paths []string
	if s, ok := a.Click.(string); ok {
		paths = append(paths, s)
	}
	if s, ok := a.DoubleClick.(string); ok {
		paths = append(paths, s)
	}
	if s, ok := a.RightClick.(string); ok {
		paths = append(paths, s)
	}
	if a.Drag != nil {
		paths = append(paths, a.Drag.From, a.Drag.To)
	}
	if a.Type != nil {
		paths = append(paths, a.Type.Into)
	}
	if s, ok := a.Wait.(string); ok {
		paths = append(paths, s)
	}
	if m, ok := a.Wait.(map[string]interface{}); ok {
		if t, ok := m["template"].(string); ok {
			paths = append(paths, t)
		}
	}
	if a.WaitGone != "" {
		paths = append(paths, a.WaitGone)
	}
	return paths
}

// applyDefaults 填充默认值
func (w *Workflow) applyDefaults() {
	if w.Settings.ElementTimeout == 0 {
		w.Settings.ElementTimeout = 10
	}
	if w.Settings.Human.Speed == 0 {
		w.Settings.Human.Speed = 1.0
	}
	if w.Settings.OnError == "" {
		w.Settings.OnError = "abort"
	}
	if w.Settings.MaxRetries == 0 {
		w.Settings.MaxRetries = 3
	}
	if w.Settings.BrowserRefreshDelay == 0 {
		w.Settings.BrowserRefreshDelay = 3000
	}
	if w.Settings.BrowserNavigationDelay == 0 {
		w.Settings.BrowserNavigationDelay = 2000
	}
	if w.Settings.BrowserPageLoadDelay == 0 {
		w.Settings.BrowserPageLoadDelay = 3000
	}
}
