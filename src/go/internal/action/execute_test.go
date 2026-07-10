package action

import (
	"errors"
	"testing"
	"time"

	"robotgo-flow/internal/config"
	"robotgo-flow/internal/geom"
)

// === ClickAction ===

func TestClickAction_Template(t *testing.T) {
	cfg := config.Action{Click: "templates/btn.png"}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}
	eng.OnFindElement("templates/btn.png", geom.Point{X: 42, Y: 17})

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("FindElement") {
		t.Error("FindElement should be called for template click")
	}
	if !eng.Called("Click") {
		t.Error("Click should be called")
	}

	c := eng.FindCall("Click")
	if c == nil {
		t.Fatal("Click call not found")
	}
	x, okX := c.Args[0].(int)
	y, okY := c.Args[1].(int)
	if !okX || !okY {
		t.Fatal("Click args should be int")
	}
	if x != 42 || y != 17 {
		t.Errorf("Click(%d, %d), want (42, 17)", x, y)
	}
}

func TestClickAction_Coord(t *testing.T) {
	cfg := config.Action{Click: map[string]interface{}{"x": 100, "y": 200}}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if eng.Called("FindElement") {
		t.Error("FindElement should NOT be called for coord click")
	}
	if !eng.Called("Click") {
		t.Error("Click should be called")
	}

	c := eng.FindCall("Click")
	x, _ := c.Args[0].(int)
	y, _ := c.Args[1].(int)
	if x != 100 || y != 200 {
		t.Errorf("Click(%d, %d), want (100, 200)", x, y)
	}
}

func TestClickAction_FindElementError(t *testing.T) {
	cfg := config.Action{Click: "templates/missing.png"}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}
	eng.SetFindError(errors.New("image not found"))

	if err := act.Execute(eng); err == nil {
		t.Fatal("Execute() should return error when FindElement fails")
	}

	if eng.Called("Click") {
		t.Error("Click should NOT be called when FindElement fails")
	}
}

// === DoubleClickAction ===

func TestDoubleClickAction_Template(t *testing.T) {
	cfg := config.Action{DoubleClick: "templates/item.png"}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}
	eng.OnFindElement("templates/item.png", geom.Point{X: 10, Y: 20})

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("FindElement") {
		t.Error("FindElement should be called")
	}
	if !eng.Called("DoubleClick") {
		t.Error("DoubleClick should be called")
	}

	c := eng.FindCall("DoubleClick")
	x, _ := c.Args[0].(int)
	y, _ := c.Args[1].(int)
	if x != 10 || y != 20 {
		t.Errorf("DoubleClick(%d, %d), want (10, 20)", x, y)
	}
}

// === RightClickAction ===

func TestRightClickAction_Template(t *testing.T) {
	cfg := config.Action{RightClick: "templates/menu.png"}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}
	eng.OnFindElement("templates/menu.png", geom.Point{X: 30, Y: 40})

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("FindElement") {
		t.Error("FindElement should be called")
	}
	if !eng.Called("RightClick") {
		t.Error("RightClick should be called")
	}

	c := eng.FindCall("RightClick")
	x, _ := c.Args[0].(int)
	y, _ := c.Args[1].(int)
	if x != 30 || y != 40 {
		t.Errorf("RightClick(%d, %d), want (30, 40)", x, y)
	}
}

// === DragAction ===

func TestDragAction_Execute(t *testing.T) {
	cfg := config.Action{Drag: &config.DragSpec{From: "templates/from.png", To: "templates/to.png"}}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}
	eng.OnFindElement("templates/from.png", geom.Point{X: 1, Y: 2})
	eng.OnFindElement("templates/to.png", geom.Point{X: 3, Y: 4})

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if eng.CallCount("FindElement") != 2 {
		t.Errorf("FindElement called %d times, want 2", eng.CallCount("FindElement"))
	}
	if !eng.Called("Drag") {
		t.Error("Drag should be called")
	}

	c := eng.FindCall("Drag")
	fromX, _ := c.Args[0].(int)
	fromY, _ := c.Args[1].(int)
	toX, _ := c.Args[2].(int)
	toY, _ := c.Args[3].(int)
	if fromX != 1 || fromY != 2 || toX != 3 || toY != 4 {
		t.Errorf("Drag(%d,%d,%d,%d), want (1,2,3,4)", fromX, fromY, toX, toY)
	}
}

// === TypeAction ===

func TestTypeAction_Execute(t *testing.T) {
	cfg := config.Action{Type: &config.TypeSpec{Into: "templates/input.png", Text: "hello"}}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}
	eng.OnFindElement("templates/input.png", geom.Point{X: 5, Y: 6})

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("FindElement") {
		t.Error("FindElement should be called")
	}
	if !eng.Called("Click") {
		t.Error("Click should be called (to focus input)")
	}
	if !eng.Called("TypeText") {
		t.Error("TypeText should be called")
	}

	tc := eng.FindCall("TypeText")
	text, _ := tc.Args[0].(string)
	if text != "hello" {
		t.Errorf("TypeText(%q), want %q", text, "hello")
	}
}

// === PressKeyAction ===

func TestPressKeyAction_Execute(t *testing.T) {
	cfg := config.Action{Press: "enter"}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("PressKey") {
		t.Error("PressKey should be called")
	}

	c := eng.FindCall("PressKey")
	key, _ := c.Args[0].(string)
	if key != "enter" {
		t.Errorf("PressKey(%q), want %q", key, "enter")
	}
}

// === PressComboAction ===

func TestPressComboAction_Execute(t *testing.T) {
	cfg := config.Action{Combo: []string{"ctrl", "c"}}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("PressCombo") {
		t.Error("PressCombo should be called")
	}

	c := eng.FindCall("PressCombo")
	if len(c.Args) != 2 {
		t.Fatalf("PressCombo args len = %d, want 2", len(c.Args))
	}
	k1, _ := c.Args[0].(string)
	k2, _ := c.Args[1].(string)
	if k1 != "ctrl" || k2 != "c" {
		t.Errorf("PressCombo(%q, %q), want (ctrl, c)", k1, k2)
	}
}

// === WaitAction ===

func TestWaitAction_Execute(t *testing.T) {
	cfg := config.Action{Wait: "templates/loading.png"}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("WaitForElement") {
		t.Error("WaitForElement should be called")
	}

	c := eng.FindCall("WaitForElement")
	tpl, _ := c.Args[0].(string)
	timeout, _ := c.Args[1].(int)
	if tpl != "templates/loading.png" {
		t.Errorf("WaitForElement template = %q, want %q", tpl, "templates/loading.png")
	}
	if timeout != 10 {
		t.Errorf("WaitForElement timeout = %d, want 10", timeout)
	}
}

// === WaitGoneAction ===

func TestWaitGoneAction_Execute(t *testing.T) {
	cfg := config.Action{WaitGone: "templates/spinner.png"}
	act, err := FromConfig(cfg, 15)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("WaitForElementGone") {
		t.Error("WaitForElementGone should be called")
	}

	c := eng.FindCall("WaitForElementGone")
	tpl, _ := c.Args[0].(string)
	timeout, _ := c.Args[1].(int)
	if tpl != "templates/spinner.png" {
		t.Errorf("WaitForElementGone template = %q, want %q", tpl, "templates/spinner.png")
	}
	if timeout != 15 {
		t.Errorf("WaitForElementGone timeout = %d, want 15", timeout)
	}
}

// === ScrollAction ===

func TestScrollAction_Down(t *testing.T) {
	cfg := config.Action{Scroll: 500}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("ScrollDown") {
		t.Error("ScrollDown should be called for positive amount")
	}
	if eng.Called("ScrollUp") {
		t.Error("ScrollUp should NOT be called for positive amount")
	}

	c := eng.FindCall("ScrollDown")
	amount, _ := c.Args[0].(int)
	if amount != 500 {
		t.Errorf("ScrollDown(%d), want 500", amount)
	}
}

func TestScrollAction_Up(t *testing.T) {
	cfg := config.Action{Scroll: -300}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("ScrollUp") {
		t.Error("ScrollUp should be called for negative amount")
	}
	if eng.Called("ScrollDown") {
		t.Error("ScrollDown should NOT be called for negative amount")
	}

	c := eng.FindCall("ScrollUp")
	amount, _ := c.Args[0].(int)
	if amount != 300 {
		t.Errorf("ScrollUp(%d), want 300 (abs)", amount)
	}
}

func TestScrollAction_Zero(t *testing.T) {
	// 直接构造（白盒测试），因为 FromConfig 拒绝 Scroll=0
	act := &ScrollAction{amount: 0}
	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if eng.Called("ScrollDown") || eng.Called("ScrollUp") {
		t.Error("ScrollDown/ScrollUp should NOT be called for zero amount")
	}
}

// === OpenURLAction ===

func TestOpenURLAction_Execute(t *testing.T) {
	cfg := config.Action{OpenURL: "https://example.com"}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("OpenURL") {
		t.Error("OpenURL should be called")
	}

	c := eng.FindCall("OpenURL")
	url, _ := c.Args[0].(string)
	if url != "https://example.com" {
		t.Errorf("OpenURL(%q), want %q", url, "https://example.com")
	}
}

// === RefreshAction ===

func TestRefreshAction_Execute(t *testing.T) {
	cfg := config.Action{Refresh: true}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("RefreshPage") {
		t.Error("RefreshPage should be called")
	}
}

// === BackAction ===

func TestBackAction_Execute(t *testing.T) {
	cfg := config.Action{Back: true}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("Back") {
		t.Error("Back should be called")
	}
}

// === ForwardAction ===

func TestForwardAction_Execute(t *testing.T) {
	cfg := config.Action{Forward: true}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("Forward") {
		t.Error("Forward should be called")
	}
}

// === SwitchTabAction ===

func TestSwitchTabAction_Execute(t *testing.T) {
	cfg := config.Action{SwitchTab: 3}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !eng.Called("SwitchTab") {
		t.Error("SwitchTab should be called")
	}

	c := eng.FindCall("SwitchTab")
	index, _ := c.Args[0].(int)
	if index != 3 {
		t.Errorf("SwitchTab(%d), want 3", index)
	}
}

// === SleepAction ===

func TestSleepAction_Execute(t *testing.T) {
	cfg := config.Action{Sleep: 0.1}
	act, err := FromConfig(cfg, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}

	eng := &MockEngine{}

	start := time.Now()
	if err := act.Execute(eng); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	elapsed := time.Since(start)

	// 宽松检查：sleep 耗时不应超过预期的 2 倍
	expected := 100 * time.Millisecond
	if elapsed > 2*expected {
		t.Errorf("sleep took too long: %v (expected ~%v)", elapsed, expected)
	}
}
