package action

import (
	"strings"
	"testing"

	"robotgo-flow/internal/config"
)

func TestFromConfig_ClickTemplate(t *testing.T) {
	act := config.Action{Click: "templates/btn.png"}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	c, ok := r.(*ClickAction)
	if !ok {
		t.Fatalf("expected *ClickAction, got %T", r)
	}
	if c.template != "templates/btn.png" {
		t.Errorf("template = %q, want %q", c.template, "templates/btn.png")
	}
	if c.coord != nil {
		t.Error("coord should be nil for template click")
	}
}

func TestFromConfig_ClickCoord(t *testing.T) {
	act := config.Action{Click: map[string]interface{}{"x": 100, "y": 200}}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	c, ok := r.(*ClickAction)
	if !ok {
		t.Fatalf("expected *ClickAction, got %T", r)
	}
	if c.coord == nil {
		t.Fatal("coord should not be nil for coord click")
	}
	if c.coord.X != 100 || c.coord.Y != 200 {
		t.Errorf("coord = (%d,%d), want (100,200)", c.coord.X, c.coord.Y)
	}
}

func TestFromConfig_DoubleClick(t *testing.T) {
	act := config.Action{DoubleClick: "templates/item.png"}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	c, ok := r.(*DoubleClickAction)
	if !ok {
		t.Fatalf("expected *DoubleClickAction, got %T", r)
	}
	if c.template != "templates/item.png" {
		t.Errorf("template = %q, want %q", c.template, "templates/item.png")
	}
}

func TestFromConfig_RightClick(t *testing.T) {
	act := config.Action{RightClick: "templates/menu.png"}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	c, ok := r.(*RightClickAction)
	if !ok {
		t.Fatalf("expected *RightClickAction, got %T", r)
	}
	if c.template != "templates/menu.png" {
		t.Errorf("template = %q, want %q", c.template, "templates/menu.png")
	}
}

func TestFromConfig_Drag(t *testing.T) {
	act := config.Action{Drag: &config.DragSpec{From: "templates/a.png", To: "templates/b.png"}}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	d, ok := r.(*DragAction)
	if !ok {
		t.Fatalf("expected *DragAction, got %T", r)
	}
	if d.fromTemplate != "templates/a.png" {
		t.Errorf("fromTemplate = %q", d.fromTemplate)
	}
	if d.toTemplate != "templates/b.png" {
		t.Errorf("toTemplate = %q", d.toTemplate)
	}
}

func TestFromConfig_Type(t *testing.T) {
	act := config.Action{Type: &config.TypeSpec{Into: "templates/input.png", Text: "hello"}}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	ta, ok := r.(*TypeAction)
	if !ok {
		t.Fatalf("expected *TypeAction, got %T", r)
	}
	if ta.template != "templates/input.png" {
		t.Errorf("template = %q", ta.template)
	}
	if ta.text != "hello" {
		t.Errorf("text = %q", ta.text)
	}
}

func TestFromConfig_PressKey(t *testing.T) {
	act := config.Action{Press: "enter"}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	p, ok := r.(*PressKeyAction)
	if !ok {
		t.Fatalf("expected *PressKeyAction, got %T", r)
	}
	if p.key != "enter" {
		t.Errorf("key = %q, want %q", p.key, "enter")
	}
}

func TestFromConfig_PressCombo(t *testing.T) {
	act := config.Action{Combo: []string{"ctrl", "c"}}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	p, ok := r.(*PressComboAction)
	if !ok {
		t.Fatalf("expected *PressComboAction, got %T", r)
	}
	if len(p.keys) != 2 || p.keys[0] != "ctrl" || p.keys[1] != "c" {
		t.Errorf("keys = %v, want [ctrl c]", p.keys)
	}
}

func TestFromConfig_WaitString(t *testing.T) {
	act := config.Action{Wait: "templates/loading.png"}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	w, ok := r.(*WaitAction)
	if !ok {
		t.Fatalf("expected *WaitAction, got %T", r)
	}
	if w.template != "templates/loading.png" {
		t.Errorf("template = %q", w.template)
	}
	if w.timeout != 10 {
		t.Errorf("timeout = %d, want 10 (default)", w.timeout)
	}
}

func TestFromConfig_WaitMap(t *testing.T) {
	act := config.Action{Wait: map[string]interface{}{"template": "templates/popup.png", "timeout": 30}}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	w, ok := r.(*WaitAction)
	if !ok {
		t.Fatalf("expected *WaitAction, got %T", r)
	}
	if w.template != "templates/popup.png" {
		t.Errorf("template = %q", w.template)
	}
	if w.timeout != 30 {
		t.Errorf("timeout = %d, want 30", w.timeout)
	}
}

func TestFromConfig_WaitGone(t *testing.T) {
	act := config.Action{WaitGone: "templates/spinner.png"}
	r, err := FromConfig(act, 15)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	w, ok := r.(*WaitGoneAction)
	if !ok {
		t.Fatalf("expected *WaitGoneAction, got %T", r)
	}
	if w.template != "templates/spinner.png" {
		t.Errorf("template = %q", w.template)
	}
	if w.timeout != 15 {
		t.Errorf("timeout = %d, want 15", w.timeout)
	}
}

func TestFromConfig_ScrollDown(t *testing.T) {
	act := config.Action{Scroll: 500}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	s, ok := r.(*ScrollAction)
	if !ok {
		t.Fatalf("expected *ScrollAction, got %T", r)
	}
	if s.amount != 500 {
		t.Errorf("amount = %d, want 500", s.amount)
	}
}

func TestFromConfig_ScrollUp(t *testing.T) {
	act := config.Action{Scroll: -300}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	s, ok := r.(*ScrollAction)
	if !ok {
		t.Fatalf("expected *ScrollAction, got %T", r)
	}
	if s.amount != -300 {
		t.Errorf("amount = %d, want -300", s.amount)
	}
}

func TestFromConfig_OpenURL(t *testing.T) {
	act := config.Action{OpenURL: "https://example.com"}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	o, ok := r.(*OpenURLAction)
	if !ok {
		t.Fatalf("expected *OpenURLAction, got %T", r)
	}
	if o.url != "https://example.com" {
		t.Errorf("url = %q", o.url)
	}
}

func TestFromConfig_Refresh(t *testing.T) {
	act := config.Action{Refresh: true}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	if _, ok := r.(*RefreshAction); !ok {
		t.Fatalf("expected *RefreshAction, got %T", r)
	}
}

func TestFromConfig_Back(t *testing.T) {
	act := config.Action{Back: true}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	if _, ok := r.(*BackAction); !ok {
		t.Fatalf("expected *BackAction, got %T", r)
	}
}

func TestFromConfig_Forward(t *testing.T) {
	act := config.Action{Forward: true}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	if _, ok := r.(*ForwardAction); !ok {
		t.Fatalf("expected *ForwardAction, got %T", r)
	}
}

func TestFromConfig_SwitchTab(t *testing.T) {
	act := config.Action{SwitchTab: 3}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	s, ok := r.(*SwitchTabAction)
	if !ok {
		t.Fatalf("expected *SwitchTabAction, got %T", r)
	}
	if s.index != 3 {
		t.Errorf("index = %d, want 3", s.index)
	}
}

func TestFromConfig_Sleep(t *testing.T) {
	act := config.Action{Sleep: 2.5}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	s, ok := r.(*SleepAction)
	if !ok {
		t.Fatalf("expected *SleepAction, got %T", r)
	}
	if s.seconds != 2.5 {
		t.Errorf("seconds = %f, want 2.5", s.seconds)
	}
}

func TestFromConfig_EmptyAction(t *testing.T) {
	act := config.Action{}
	_, err := FromConfig(act, 10)
	if err == nil {
		t.Error("FromConfig() should fail for empty action")
	}
}

func TestFromConfig_ClickInvalidType(t *testing.T) {
	act := config.Action{Click: 42} // int is invalid
	_, err := FromConfig(act, 10)
	if err == nil {
		t.Error("FromConfig() should fail for invalid click type")
	}
	if !strings.Contains(err.Error(), "需要字符串或 {x,y} 格式") {
		t.Errorf("error = %v, should mention type issue", err)
	}
}

func TestFromConfig_WaitMapMissingTemplate(t *testing.T) {
	act := config.Action{Wait: map[string]interface{}{"timeout": 30}}
	_, err := FromConfig(act, 10)
	if err == nil {
		t.Error("FromConfig() should fail when wait map has no template")
	}
}

func TestFromConfig_DoubleClickCoord(t *testing.T) {
	act := config.Action{DoubleClick: map[string]interface{}{"x": 50, "y": 60}}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	c, ok := r.(*DoubleClickAction)
	if !ok {
		t.Fatalf("expected *DoubleClickAction, got %T", r)
	}
	if c.coord == nil {
		t.Fatal("coord should not be nil")
	}
	if c.coord.X != 50 || c.coord.Y != 60 {
		t.Errorf("coord = (%d,%d), want (50,60)", c.coord.X, c.coord.Y)
	}
}

func TestFromConfig_RightClickCoord(t *testing.T) {
	act := config.Action{RightClick: map[string]interface{}{"x": 10, "y": 20}}
	r, err := FromConfig(act, 10)
	if err != nil {
		t.Fatalf("FromConfig() error = %v", err)
	}
	c, ok := r.(*RightClickAction)
	if !ok {
		t.Fatalf("expected *RightClickAction, got %T", r)
	}
	if c.coord == nil {
		t.Fatal("coord should not be nil")
	}
	if c.coord.X != 10 || c.coord.Y != 20 {
		t.Errorf("coord = (%d,%d), want (10,20)", c.coord.X, c.coord.Y)
	}
}
