package action

import (
	"testing"
)

func TestPromptAction_DoesNotPanic(t *testing.T) {
	eng := &MockEngine{}
	act := &PromptAction{title: "Test", message: "Enter value", into: "", mask: false}

	// 在 CI/无头环境下，交互输入返回空字符串，因此 Execute 返回错误，
	// 但任何情况下都不应 panic。
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PromptAction panicked: %v", r)
		}
	}()
	err := act.Execute(eng)
	if err == nil {
		t.Log("PromptAction 执行成功（当前环境可交互）")
	} else {
		t.Logf("PromptAction 在无头环境下按预期失败: %v", err)
	}
}

func TestConfirmAction_Executes(t *testing.T) {
	eng := &MockEngine{}
	act := &ConfirmAction{title: "Confirm?", message: "Are you sure?"}

	// 在 CI/无头环境下，确认框返回 false，因此 Execute 返回错误，
	// 但任何情况下都不应 panic。
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ConfirmAction panicked: %v", r)
		}
	}()
	err := act.Execute(eng)
	if err == nil {
		t.Log("ConfirmAction 执行成功（当前环境可交互）")
	} else {
		t.Logf("ConfirmAction 在无头环境下按预期失败: %v", err)
	}
}

func TestNotifyAction_Executes(t *testing.T) {
	eng := &MockEngine{}
	// 测试 duration > 0 时（非阻塞通知路径）
	t.Run("with duration", func(t *testing.T) {
		act := &NotifyAction{title: "Note", message: "Something happened", duration: 3}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("NotifyAction panicked: %v", r)
			}
		}()
		err := act.Execute(eng)
		if err != nil {
			t.Errorf("NotifyAction should not error: %v", err)
		}
	})
	// 测试 duration == 0 时（模态路径）
	t.Run("without duration", func(t *testing.T) {
		act := &NotifyAction{title: "Note", message: "Something happened"}
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("NotifyAction panicked: %v", r)
			}
		}()
		err := act.Execute(eng)
		if err != nil {
			t.Errorf("NotifyAction (no duration) should not error: %v", err)
		}
	})
}
