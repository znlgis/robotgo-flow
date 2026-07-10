package config

import (
	"testing"
)

// =============================================================================
// isEmpty
// =============================================================================

func TestAction_isEmpty(t *testing.T) {
	tests := []struct {
		name   string
		action Action
		want   bool
	}{
		{
			name:   "empty action (all zero values)",
			action: Action{},
			want:   true,
		},
		// 每个字段单独设置 → false
		{
			name:   "Click set (string)",
			action: Action{Click: "btn.png"},
			want:   false,
		},
		{
			name:   "DoubleClick set (string)",
			action: Action{DoubleClick: "btn.png"},
			want:   false,
		},
		{
			name:   "RightClick set (string)",
			action: Action{RightClick: "btn.png"},
			want:   false,
		},
		{
			name:   "Drag set",
			action: Action{Drag: &DragSpec{From: "a.png", To: "b.png"}},
			want:   false,
		},
		{
			name:   "Type set",
			action: Action{Type: &TypeSpec{Into: "input.png", Text: "hello"}},
			want:   false,
		},
		{
			name:   "Press set",
			action: Action{Press: "enter"},
			want:   false,
		},
		{
			name:   "Combo set",
			action: Action{Combo: []string{"ctrl", "c"}},
			want:   false,
		},
		{
			name:   "Wait set (string)",
			action: Action{Wait: "loading.png"},
			want:   false,
		},
		{
			name:   "WaitGone set",
			action: Action{WaitGone: "popup.png"},
			want:   false,
		},
		{
			name:   "Scroll set (non-zero)",
			action: Action{Scroll: 3},
			want:   false,
		},
		{
			name:   "Scroll set (negative)",
			action: Action{Scroll: -1},
			want:   false,
		},
		{
			name:   "OpenURL set",
			action: Action{OpenURL: "https://example.com"},
			want:   false,
		},
		{
			name:   "Refresh set (true)",
			action: Action{Refresh: true},
			want:   false,
		},
		{
			name:   "Back set (true)",
			action: Action{Back: true},
			want:   false,
		},
		{
			name:   "Forward set (true)",
			action: Action{Forward: true},
			want:   false,
		},
		{
			name:   "SwitchTab set (positive)",
			action: Action{SwitchTab: 2},
			want:   false,
		},
		{
			name:   "Sleep set (positive)",
			action: Action{Sleep: 1.5},
			want:   false,
		},
		// false 布尔字段不应算作已设置
		{
			name:   "Refresh false (should be empty)",
			action: Action{Refresh: false},
			want:   true,
		},
		{
			name:   "Back false (should be empty)",
			action: Action{Back: false},
			want:   true,
		},
		{
			name:   "Forward false (should be empty)",
			action: Action{Forward: false},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.action.isEmpty()
			if got != tt.want {
				t.Errorf("isEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// validateSingleAction
// =============================================================================

func TestAction_validateSingleAction(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr bool
	}{
		// 恰好一个字段设置 → 无错误
		{
			name:    "one field: Click",
			action:  Action{Click: "btn.png"},
			wantErr: false,
		},
		{
			name:    "one field: DoubleClick",
			action:  Action{DoubleClick: "btn.png"},
			wantErr: false,
		},
		{
			name:    "one field: RightClick",
			action:  Action{RightClick: "btn.png"},
			wantErr: false,
		},
		{
			name:    "one field: Drag",
			action:  Action{Drag: &DragSpec{From: "a.png", To: "b.png"}},
			wantErr: false,
		},
		{
			name:    "one field: Type",
			action:  Action{Type: &TypeSpec{Into: "input.png", Text: "hello"}},
			wantErr: false,
		},
		{
			name:    "one field: Press",
			action:  Action{Press: "enter"},
			wantErr: false,
		},
		{
			name:    "one field: Combo",
			action:  Action{Combo: []string{"ctrl", "c"}},
			wantErr: false,
		},
		{
			name:    "one field: Wait",
			action:  Action{Wait: "loading.png"},
			wantErr: false,
		},
		{
			name:    "one field: WaitGone",
			action:  Action{WaitGone: "popup.png"},
			wantErr: false,
		},
		{
			name:    "one field: Scroll",
			action:  Action{Scroll: 5},
			wantErr: false,
		},
		{
			name:    "one field: OpenURL",
			action:  Action{OpenURL: "https://example.com"},
			wantErr: false,
		},
		{
			name:    "one field: Refresh",
			action:  Action{Refresh: true},
			wantErr: false,
		},
		{
			name:    "one field: Back",
			action:  Action{Back: true},
			wantErr: false,
		},
		{
			name:    "one field: Forward",
			action:  Action{Forward: true},
			wantErr: false,
		},
		{
			name:    "one field: SwitchTab",
			action:  Action{SwitchTab: 3},
			wantErr: false,
		},
		{
			name:    "one field: Sleep",
			action:  Action{Sleep: 2.0},
			wantErr: false,
		},
		// 零字段 → 无错误（isEmpty 单独处理）
		{
			name:    "zero fields (empty action)",
			action:  Action{},
			wantErr: false,
		},
		// 两个字段设置 → 错误
		{
			name:    "two fields: Click + Sleep",
			action:  Action{Click: "btn.png", Sleep: 1.0},
			wantErr: true,
		},
		{
			name:    "two fields: Press + Combo",
			action:  Action{Press: "enter", Combo: []string{"ctrl", "v"}},
			wantErr: true,
		},
		{
			name:    "two fields: Scroll + OpenURL",
			action:  Action{Scroll: 3, OpenURL: "https://example.com"},
			wantErr: true,
		},
		{
			name:    "two fields: Refresh + Back",
			action:  Action{Refresh: true, Back: true},
			wantErr: true,
		},
		// 多个字段设置 → 错误
		{
			name: "three fields: Click + Press + Sleep",
			action: Action{
				Click: "btn.png",
				Press: "enter",
				Sleep: 1.0,
			},
			wantErr: true,
		},
		{
			name: "four fields: Click + Drag + Type + Wait",
			action: Action{
				Click: "btn.png",
				Drag:  &DragSpec{From: "a.png", To: "b.png"},
				Type:  &TypeSpec{Into: "input.png", Text: "hello"},
				Wait:  "loading.png",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.validateSingleAction()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSingleAction() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// =============================================================================
// validateMapFields
// =============================================================================

func TestAction_validateMapFields(t *testing.T) {
	tests := []struct {
		name    string
		action  Action
		wantErr bool
		errMsg  string
	}{
		// 拖拽场景
		{
			name:    "Drag with both From and To set",
			action:  Action{Drag: &DragSpec{From: "a.png", To: "b.png"}},
			wantErr: false,
		},
		{
			name:    "Drag with empty From",
			action:  Action{Drag: &DragSpec{From: "", To: "b.png"}},
			wantErr: true,
			errMsg:  "拖拽起点不能为空",
		},
		{
			name:    "Drag with empty To",
			action:  Action{Drag: &DragSpec{From: "a.png", To: ""}},
			wantErr: true,
			errMsg:  "拖拽终点不能为空",
		},
		{
			name:    "Drag with both empty",
			action:  Action{Drag: &DragSpec{From: "", To: ""}},
			wantErr: true,
			errMsg:  "拖拽起点不能为空",
		},
		// 输入场景
		{
			name:    "Type with both Into and Text set",
			action:  Action{Type: &TypeSpec{Into: "input.png", Text: "hello"}},
			wantErr: false,
		},
		{
			name:    "Type with empty Into",
			action:  Action{Type: &TypeSpec{Into: "", Text: "hello"}},
			wantErr: true,
			errMsg:  "输入目标不能为空",
		},
		{
			name:    "Type with empty Text",
			action:  Action{Type: &TypeSpec{Into: "input.png", Text: ""}},
			wantErr: true,
			errMsg:  "输入文本不能为空",
		},
		{
			name:    "Type with both empty",
			action:  Action{Type: &TypeSpec{Into: "", Text: ""}},
			wantErr: true,
			errMsg:  "输入目标不能为空",
		},
		// 无 Drag 或 Type → 无错误（nil 场景）
		{
			name:    "no Drag or Type (Click only)",
			action:  Action{Click: "btn.png"},
			wantErr: false,
		},
		{
			name:    "no Drag or Type (Press only)",
			action:  Action{Press: "enter"},
			wantErr: false,
		},
		{
			name:    "no Drag or Type (empty action)",
			action:  Action{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.action.validateMapFields()
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMapFields() error = %v, wantErr = %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.errMsg {
				t.Errorf("validateMapFields() error = %q, want %q", err.Error(), tt.errMsg)
			}
		})
	}
}

// =============================================================================
// applyDefaults
// =============================================================================

func TestWorkflow_applyDefaults(t *testing.T) {
	tests := []struct {
		name     string
		workflow Workflow
		wantTO   int
		wantSpd  float64
	}{
		{
			name:     "zero timeout → 10, zero speed → 1.0",
			workflow: Workflow{},
			wantTO:   10,
			wantSpd:  1.0,
		},
		{
			name: "zero timeout only, speed already set",
			workflow: Workflow{
				Settings: Settings{
					ElementTimeout: 0,
					Human:          HumanConf{Speed: 2.0},
				},
			},
			wantTO:  10,
			wantSpd: 2.0,
		},
		{
			name: "timeout already set, zero speed",
			workflow: Workflow{
				Settings: Settings{
					ElementTimeout: 30,
					Human:          HumanConf{Speed: 0},
				},
			},
			wantTO:  30,
			wantSpd: 1.0,
		},
		{
			name: "both values already set — preserved",
			workflow: Workflow{
				Settings: Settings{
					ElementTimeout: 30,
					Human:          HumanConf{Speed: 2.5},
				},
			},
			wantTO:  30,
			wantSpd: 2.5,
		},
		{
			name: "human disabled, zero speed — still gets default 1.0",
			workflow: Workflow{
				Settings: Settings{
					ElementTimeout: 15,
					Human:          HumanConf{Enabled: false, Speed: 0},
				},
			},
			wantTO:  15,
			wantSpd: 1.0,
		},
		{
			name: "human disabled, non-zero speed — speed untouched",
			workflow: Workflow{
				Settings: Settings{
					ElementTimeout: 15,
					Human:          HumanConf{Enabled: false, Speed: 3.0},
				},
			},
			wantTO:  15,
			wantSpd: 3.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wf := tt.workflow // copy
			wf.applyDefaults()

			if wf.Settings.ElementTimeout != tt.wantTO {
				t.Errorf("ElementTimeout = %d, want %d", wf.Settings.ElementTimeout, tt.wantTO)
			}
			if wf.Settings.Human.Speed != tt.wantSpd {
				t.Errorf("Human.Speed = %f, want %f", wf.Settings.Human.Speed, tt.wantSpd)
			}
		})
	}
}

// =============================================================================
// templatePaths
// =============================================================================

func TestAction_templatePaths(t *testing.T) {
	tests := []struct {
		name   string
		action Action
		want   []string
	}{
		{
			name:   "empty action — no paths",
			action: Action{},
			want:   nil,
		},
		{
			name:   "Click 字符串模板",
			action: Action{Click: "btn.png"},
			want:   []string{"btn.png"},
		},
		{
			name:   "Click map（坐标）— 无路径",
			action: Action{Click: map[string]interface{}{"x": 100, "y": 200}},
			want:   nil,
		},
		{
			name:   "Click int — 无路径",
			action: Action{Click: 42},
			want:   nil,
		},
		{
			name:   "DoubleClick 字符串模板",
			action: Action{DoubleClick: "btn_double.png"},
			want:   []string{"btn_double.png"},
		},
		{
			name:   "DoubleClick map（坐标）— 无路径",
			action: Action{DoubleClick: map[string]interface{}{"x": 100, "y": 200}},
			want:   nil,
		},
		{
			name:   "RightClick 字符串模板",
			action: Action{RightClick: "btn_right.png"},
			want:   []string{"btn_right.png"},
		},
		{
			name:   "RightClick map（坐标）— 无路径",
			action: Action{RightClick: map[string]interface{}{"x": 100, "y": 200}},
			want:   nil,
		},
		{
			name:   "Drag 含起点终点",
			action: Action{Drag: &DragSpec{From: "start.png", To: "end.png"}},
			want:   []string{"start.png", "end.png"},
		},
		{
			name:   "Drag 空起点/终点 — 仍包含在路径中",
			action: Action{Drag: &DragSpec{From: "", To: ""}},
			want:   []string{"", ""},
		},
		{
			name:   "Type 含 Into",
			action: Action{Type: &TypeSpec{Into: "input.png", Text: "hello"}},
			want:   []string{"input.png"},
		},
		{
			name:   "Type 空 Into",
			action: Action{Type: &TypeSpec{Into: "", Text: "hello"}},
			want:   []string{""},
		},
		{
			name:   "Wait 字符串",
			action: Action{Wait: "loading.png"},
			want:   []string{"loading.png"},
		},
		{
			name:   "Wait map 含 template 键",
			action: Action{Wait: map[string]interface{}{"template": "popup.png", "timeout": 5}},
			want:   []string{"popup.png"},
		},
		{
			name:   "Wait map 不含 template 键",
			action: Action{Wait: map[string]interface{}{"timeout": 5}},
			want:   nil,
		},
		{
			name:   "Wait int — 无路径",
			action: Action{Wait: 10},
			want:   nil,
		},
		{
			name:   "WaitGone 已设置",
			action: Action{WaitGone: "gone.png"},
			want:   []string{"gone.png"},
		},
		{
			name:   "WaitGone 空字符串 — 不包含",
			action: Action{WaitGone: ""},
			want:   nil,
		},
		{
			name: "多模板字段 — 收集全部路径",
			action: Action{
				Click:       "btn.png",
				DoubleClick: "btn2.png",
				RightClick:  "btn3.png",
				Drag:        &DragSpec{From: "from.png", To: "to.png"},
				Type:        &TypeSpec{Into: "input.png", Text: "hello"},
				Wait:        "loading.png",
				WaitGone:    "gone.png",
			},
			want: []string{"btn.png", "btn2.png", "btn3.png", "from.png", "to.png", "input.png", "loading.png", "gone.png"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.action.templatePaths()

			if tt.want == nil && got != nil {
				t.Errorf("templatePaths() = %v, want nil", got)
				return
			}
			if tt.want == nil && got == nil {
				return
			}

			if len(got) != len(tt.want) {
				t.Errorf("templatePaths() len = %d, want %d; got = %v, want = %v",
					len(got), len(tt.want), got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("templatePaths()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
