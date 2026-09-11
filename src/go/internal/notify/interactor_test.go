package notify

import (
	"errors"
	"strings"
	"testing"
)

// fakeInteractor 记录调用并返回预设值，用于验证 Interactor 注入机制。
type fakeInteractor struct {
	inputVal   string
	inputErr   error
	confirmVal bool
	confirmErr error
	inputs     []string
	confirms   []string
}

func (f *fakeInteractor) Input(label, placeholder string, mask bool) (string, error) {
	f.inputs = append(f.inputs, label)
	return f.inputVal, f.inputErr
}

func (f *fakeInteractor) Confirm(title, message string) (bool, error) {
	f.confirms = append(f.confirms, title)
	return f.confirmVal, f.confirmErr
}

func TestSetInteractor_ReplacesDefault(t *testing.T) {
	t.Cleanup(func() { SetInteractor(nil) })

	fake := &fakeInteractor{inputVal: "hunter2", confirmVal: true}
	SetInteractor(fake)

	if InteractorEnabled() {
		t.Error("注入自定义实现后 InteractorEnabled 应为 false")
	}

	got, err := InputBoxStd("密码", "请输入", true)
	if err != nil {
		t.Fatalf("InputBoxStd() error = %v", err)
	}
	if got != "hunter2" {
		t.Errorf("InputBoxStd() = %q, want %q", got, "hunter2")
	}

	ok, err := ConfirmBoxStd("确认", "继续?")
	if err != nil {
		t.Fatalf("ConfirmBoxStd() error = %v", err)
	}
	if !ok {
		t.Error("ConfirmBoxStd() = false, want true")
	}

	if len(fake.inputs) != 1 || fake.inputs[0] != "密码" {
		t.Errorf("Input 调用参数错误: %v", fake.inputs)
	}
	if len(fake.confirms) != 1 || fake.confirms[0] != "确认" {
		t.Errorf("Confirm 调用参数错误: %v", fake.confirms)
	}
}

func TestSetInteractor_NilRestoresStdin(t *testing.T) {
	SetInteractor(&fakeInteractor{})
	SetInteractor(nil)

	if !InteractorEnabled() {
		t.Error("SetInteractor(nil) 应恢复默认 stdin 实现")
	}
}

func TestDisableInteraction_FailsFast(t *testing.T) {
	t.Cleanup(func() { SetInteractor(nil) })

	DisableInteraction("单元测试")

	if _, err := InputBoxStd("标签", "", false); err == nil {
		t.Error("禁用交互后 InputBoxStd 应返回错误")
	} else if !strings.Contains(err.Error(), "不支持") {
		t.Errorf("错误信息应说明原因, got %v", err)
	}

	if _, err := ConfirmBoxStd("确认", "继续?"); err == nil {
		t.Error("禁用交互后 ConfirmBoxStd 应返回错误")
	}
}

func TestInteractor_PropagatesError(t *testing.T) {
	t.Cleanup(func() { SetInteractor(nil) })

	wantErr := errors.New("输入已结束")
	SetInteractor(&fakeInteractor{inputErr: wantErr, confirmErr: wantErr})

	if _, err := InputBoxStd("标签", "", false); !errors.Is(err, wantErr) {
		t.Errorf("InputBoxStd() error = %v, want %v", err, wantErr)
	}
	if _, err := ConfirmBoxStd("确认", "继续?"); !errors.Is(err, wantErr) {
		t.Errorf("ConfirmBoxStd() error = %v, want %v", err, wantErr)
	}
}
