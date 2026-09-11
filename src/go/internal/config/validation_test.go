package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// baseWorkflow 返回一个可通过校验的最小工作流（含默认值）。
func baseWorkflow() *Workflow {
	wf := &Workflow{
		Name:  "校验测试",
		Steps: []Step{{Name: "步骤1", Actions: []Action{{Sleep: 1}}}},
	}
	wf.applyDefaults()
	return wf
}

func TestValidate_OnErrorEnum(t *testing.T) {
	for _, valid := range []string{"abort", "skip", "retry"} {
		wf := baseWorkflow()
		wf.Settings.OnError = valid
		if err := wf.Validate(t.TempDir()); err != nil {
			t.Errorf("on_error=%q 应通过校验, got %v", valid, err)
		}
	}

	for _, invalid := range []string{"Retry", "ignore", "continue"} {
		wf := baseWorkflow()
		wf.Settings.OnError = invalid
		err := wf.Validate(t.TempDir())
		if err == nil {
			t.Fatalf("on_error=%q 应校验失败", invalid)
		}
		if !strings.Contains(err.Error(), "on_error") {
			t.Errorf("错误信息应包含字段名, got %v", err)
		}
	}
}

func TestValidate_MaxRetries(t *testing.T) {
	wf := baseWorkflow()
	wf.Settings.MaxRetries = 0
	if err := wf.Validate(t.TempDir()); err == nil {
		t.Error("max_retries=0 应校验失败")
	}

	wf = baseWorkflow()
	wf.Settings.MaxRetries = 3
	if err := wf.Validate(t.TempDir()); err != nil {
		t.Errorf("max_retries=3 应通过校验, got %v", err)
	}
}

func TestValidate_Inputs(t *testing.T) {
	tests := []struct {
		name    string
		inputs  []InputSpec
		wantErr string
	}{
		{
			name:   "合法输入",
			inputs: []InputSpec{{Name: "username", Label: "用户名"}},
		},
		{
			name:    "缺少 name",
			inputs:  []InputSpec{{Label: "用户名"}},
			wantErr: "name 不能为空",
		},
		{
			name:    "name 含首尾空白",
			inputs:  []InputSpec{{Name: " user ", Label: "用户名"}},
			wantErr: "首尾空白",
		},
		{
			name:    "name 重复",
			inputs:  []InputSpec{{Name: "user", Label: "A"}, {Name: "user", Label: "B"}},
			wantErr: "重复",
		},
		{
			name:    "缺少 label",
			inputs:  []InputSpec{{Name: "user"}},
			wantErr: "label 不能为空",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := baseWorkflow()
			wf.Inputs = tt.inputs
			err := wf.Validate(t.TempDir())
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("应通过校验, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("应校验失败并包含 %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("错误信息 = %v, want 包含 %q", err, tt.wantErr)
			}
		})
	}
}

// prompt.intO 引用的模板此前不会被校验，导致运行时才失败。
func TestValidate_PromptIntoTemplate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "input.png"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	wf := baseWorkflow()
	wf.Steps[0].Actions = []Action{{Prompt: &PromptSpec{Title: "标题", Message: "消息", Into: "input.png"}}}
	if err := wf.Validate(dir); err != nil {
		t.Fatalf("已存在的 prompt.into 模板应通过校验, got %v", err)
	}

	wf.Steps[0].Actions = []Action{{Prompt: &PromptSpec{Title: "标题", Message: "消息", Into: "missing.png"}}}
	err := wf.Validate(dir)
	if err == nil || !strings.Contains(err.Error(), "模板未找到") {
		t.Errorf("缺失的 prompt.into 模板应校验失败, got %v", err)
	}
}

// click 支持 {template: 路径} 写法，其模板同样需要参与存在性校验。
func TestValidate_ClickMapTemplate(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "btn.png"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	wf := baseWorkflow()
	wf.Steps[0].Actions = []Action{{Click: map[string]interface{}{"template": "btn.png"}}}
	if err := wf.Validate(dir); err != nil {
		t.Fatalf("模板存在时应通过校验, got %v", err)
	}

	wf.Steps[0].Actions = []Action{{Click: map[string]interface{}{"template": "missing.png"}}}
	if err := wf.Validate(dir); err == nil || !strings.Contains(err.Error(), "模板未找到") {
		t.Errorf("模板缺失时应校验失败, got %v", err)
	}

	// 坐标写法不应被当作模板路径
	wf.Steps[0].Actions = []Action{{Click: map[string]interface{}{"x": 10, "y": 20}}}
	if err := wf.Validate(dir); err != nil {
		t.Errorf("坐标写法应通过校验, got %v", err)
	}
}

func TestLoad_ClickTemplateMapForm(t *testing.T) {
	yaml := `
name: "map 模板写法"
steps:
  - name: "步骤1"
    actions:
      - click: {template: "templates/btn.png"}
      - wait: {template: "templates/btn.png", timeout: 5}
`
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "templates"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templates", "btn.png"), []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	path := filepath.Join(dir, "wf.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	wf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if got := wf.Steps[0].Actions[0].Label(); !strings.Contains(got, "templates/btn.png") {
		t.Errorf("click map 标签 = %q, want 含模板路径", got)
	}
}
