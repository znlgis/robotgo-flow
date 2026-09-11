package engine

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestResolveTemplate_RelativeAndAbsolute(t *testing.T) {
	dir := t.TempDir()
	tplDir := filepath.Join(dir, "templates")
	if err := os.MkdirAll(tplDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	rel := filepath.Join(tplDir, "btn.png")
	if err := os.WriteFile(rel, []byte("x"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	e := NewEngine(tplDir, filepath.Join(dir, "shots"))

	// 相对路径以 templateDir 为基准解析
	got, err := e.resolveTemplate("btn.png")
	if err != nil {
		t.Fatalf("resolveTemplate(相对路径) error = %v", err)
	}
	if got != filepath.Clean(rel) {
		t.Errorf("resolveTemplate() = %q, want %q", got, filepath.Clean(rel))
	}

	// 绝对路径原样使用（不再与 templateDir 拼接）
	got, err = e.resolveTemplate(rel)
	if err != nil {
		t.Fatalf("resolveTemplate(绝对路径) error = %v", err)
	}
	if got != filepath.Clean(rel) {
		t.Errorf("resolveTemplate() = %q, want %q", got, filepath.Clean(rel))
	}
}

func TestResolveTemplate_NotFound(t *testing.T) {
	e := NewEngine(t.TempDir(), "")

	_, err := e.resolveTemplate("missing.png")
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("error = %v, want ErrTemplateNotFound", err)
	}
	if !IsTemplateError(err) {
		t.Error("IsTemplateError() = false, want true")
	}

	_, err = e.resolveTemplate("")
	if !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("空模板名 error = %v, want ErrTemplateNotFound", err)
	}
}

func TestIsTemplateError_OtherErrors(t *testing.T) {
	if IsTemplateError(errors.New("屏幕未找到")) {
		t.Error("普通错误不应被判定为模板错误")
	}
	if IsTemplateError(nil) {
		t.Error("nil 不应被判定为模板错误")
	}
}

// WaitForElement / WaitForElementGone 对缺失模板必须立即失败，
// 而不是轮询等待到超时（旧实现会把“模板不存在”误判为“元素已消失”）。
func TestWaitForElement_MissingTemplateFailsFast(t *testing.T) {
	e := NewEngine(t.TempDir(), "")

	start := time.Now()
	err := e.WaitForElement("missing.png", 5)
	elapsed := time.Since(start)

	if !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("WaitForElement() error = %v, want ErrTemplateNotFound", err)
	}
	if elapsed > time.Second {
		t.Errorf("应在解析模板阶段立即失败，实际耗时 %v", elapsed)
	}
}

func TestWaitForElementGone_MissingTemplateFailsFast(t *testing.T) {
	e := NewEngine(t.TempDir(), "")

	start := time.Now()
	err := e.WaitForElementGone("missing.png", 5)
	elapsed := time.Since(start)

	if !errors.Is(err, ErrTemplateNotFound) {
		t.Fatalf("WaitForElementGone() error = %v, want ErrTemplateNotFound", err)
	}
	if elapsed > time.Second {
		t.Errorf("应在解析模板阶段立即失败，实际耗时 %v", elapsed)
	}
}
