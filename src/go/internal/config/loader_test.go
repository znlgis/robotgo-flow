package config

import (
	"os"
	"path/filepath"
	"testing"
)

// writeTempYAML 创建临时 YAML 文件用于测试，返回文件路径和清理函数
func writeTempYAML(t *testing.T, name string, content string) (string, func()) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}
	return path, func() {}
}

func TestLoad_ValidWorkflow(t *testing.T) {
	yaml := `
name: "测试工作流"
description: "测试描述"
settings:
  element_timeout: 15
  human:
    enabled: true
    speed: 1.5
    mistake_rate: 0.05
steps:
  - name: "步骤1"
    actions:
      - press: "enter"
      - sleep: 2
`
	path, _ := writeTempYAML(t, "test.yaml", yaml)

	wf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if wf.Name != "测试工作流" {
		t.Errorf("Name = %q, want %q", wf.Name, "测试工作流")
	}
	if wf.Description != "测试描述" {
		t.Errorf("Description = %q, want %q", wf.Description, "测试描述")
	}
	if wf.Settings.ElementTimeout != 15 {
		t.Errorf("ElementTimeout = %d, want 15", wf.Settings.ElementTimeout)
	}
	if wf.Settings.Human.Speed != 1.5 {
		t.Errorf("Human.Speed = %f, want 1.5", wf.Settings.Human.Speed)
	}
	if wf.Settings.Human.MistakeRate != 0.05 {
		t.Errorf("Human.MistakeRate = %f, want 0.05", wf.Settings.Human.MistakeRate)
	}
	if len(wf.Steps) != 1 {
		t.Fatalf("len(Steps) = %d, want 1", len(wf.Steps))
	}
	if len(wf.Steps[0].Actions) != 2 {
		t.Fatalf("len(Actions) = %d, want 2", len(wf.Steps[0].Actions))
	}
}

func TestLoad_Defaults(t *testing.T) {
	yaml := `
name: "默认值测试"
settings:
  element_timeout: 0
steps:
  - name: "步骤1"
    actions:
      - sleep: 1
`
	path, _ := writeTempYAML(t, "test.yaml", yaml)

	wf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if wf.Settings.ElementTimeout != 10 {
		t.Errorf("default ElementTimeout = %d, want 10", wf.Settings.ElementTimeout)
	}
	if wf.Settings.Human.Speed != 1.0 {
		t.Errorf("default Human.Speed = %f, want 1.0", wf.Settings.Human.Speed)
	}
}

func TestLoad_MissingName(t *testing.T) {
	yaml := `
settings:
  element_timeout: 10
steps:
  - name: "步骤1"
    actions:
      - sleep: 1
`
	path, _ := writeTempYAML(t, "test.yaml", yaml)

	_, err := Load(path)
	if err == nil {
		t.Error("Load() should fail when name is missing")
	}
}

func TestLoad_MissingSteps(t *testing.T) {
	yaml := `
name: "无步骤"
settings:
  element_timeout: 10
`
	path, _ := writeTempYAML(t, "test.yaml", yaml)

	_, err := Load(path)
	if err == nil {
		t.Error("Load() should fail when steps are missing")
	}
}

func TestLoad_EmptyStepName(t *testing.T) {
	yaml := `
name: "空步骤名"
settings:
  element_timeout: 10
steps:
  - name: ""
    actions:
      - sleep: 1
`
	path, _ := writeTempYAML(t, "test.yaml", yaml)

	_, err := Load(path)
	if err == nil {
		t.Error("Load() should fail with empty step name")
	}
}

func TestLoad_NoActions(t *testing.T) {
	yaml := `
name: "无动作"
settings:
  element_timeout: 10
steps:
  - name: "步骤1"
    actions: []
`
	path, _ := writeTempYAML(t, "test.yaml", yaml)

	_, err := Load(path)
	if err == nil {
		t.Error("Load() should fail with empty actions")
	}
}

func TestLoad_MultipleActionTypes(t *testing.T) {
	yaml := `
name: "重复动作"
settings:
  element_timeout: 10
steps:
  - name: "步骤1"
    actions:
      - click: "btn.png"
        sleep: 1
`
	path, _ := writeTempYAML(t, "test.yaml", yaml)

	_, err := Load(path)
	if err == nil {
		t.Error("Load() should fail when an action has multiple types")
	}
}

func TestLoad_InvalidHumanSpeed(t *testing.T) {
	yaml := `
name: "无效速度"
settings:
  element_timeout: 10
  human:
    enabled: true
    speed: 10.0
    mistake_rate: 0.05
steps:
  - name: "步骤1"
    actions:
      - sleep: 1
`
	path, _ := writeTempYAML(t, "test.yaml", yaml)

	_, err := Load(path)
	if err == nil {
		t.Error("Load() should fail with invalid human speed")
	}
}

func TestLoad_InvalidMistakeRate(t *testing.T) {
	yaml := `
name: "无效错误率"
settings:
  element_timeout: 10
  human:
    enabled: true
    speed: 1.0
    mistake_rate: 2.0
steps:
  - name: "步骤1"
    actions:
      - sleep: 1
`
	path, _ := writeTempYAML(t, "test.yaml", yaml)

	_, err := Load(path)
	if err == nil {
		t.Error("Load() should fail with invalid mistake_rate")
	}
}

func TestLoad_NegativeTimeout(t *testing.T) {
	yaml := `
name: "负超时"
settings:
  element_timeout: 0
steps:
  - name: "步骤1"
    actions:
      - sleep: 1
`
	path, _ := writeTempYAML(t, "test.yaml", yaml)

	wf, err := Load(path)
	if err != nil {
		t.Fatalf("Load() should apply default for zero timeout: %v", err)
	}
	// 默认值应该被应用（0 → 10）
	if wf.Settings.ElementTimeout != 10 {
		t.Errorf("ElementTimeout = %d, want 10 (default applied for 0)", wf.Settings.ElementTimeout)
	}
}
