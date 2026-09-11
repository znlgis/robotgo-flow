package main

import (
	"flag"
	"reflect"
	"testing"
)

// newRunLikeFlagSet 构造与 run/serve 子命令同形态的 FlagSet（int/string/bool 各一），
// 用于测试 reorderFlags 对各类 flag 的处理。
func newRunLikeFlagSet(name string) (*flag.FlagSet, *int, *string, *bool) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	from := fs.Int("from", 1, "从指定步骤开始执行 (1-indexed)")
	out := fs.String("out", "", "截图输出目录")
	debug := fs.Bool("debug", false, "调试模式")
	return fs, from, out, debug
}

// TestReorderFlags 验证把位置参数之后的 flag 重排到最前，
// 使 `run workflow.yaml --from 3` 与 `run --from 3 workflow.yaml` 等价。
func TestReorderFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "flags 在前_保持原顺序",
			args: []string{"--from", "3", "--out", "shots", "--debug", "workflow.yaml"},
			want: []string{"--from", "3", "--out", "shots", "--debug", "workflow.yaml"},
		},
		{
			name: "README 写法_int 与 bool flag 在位置参数之后",
			args: []string{"workflow.yaml", "--from", "3", "--debug"},
			want: []string{"--from", "3", "--debug", "workflow.yaml"},
		},
		{
			name: "等号形式在位置参数之后",
			args: []string{"workflow.yaml", "--from=3"},
			want: []string{"--from=3", "workflow.yaml"},
		},
		{
			name: "需要值的 string flag 在位置参数之后_值一并前移",
			args: []string{"workflow.yaml", "--out", "shots"},
			want: []string{"--out", "shots", "workflow.yaml"},
		},
		{
			name: "位置参数夹在 flags 中间",
			args: []string{"--from", "3", "workflow.yaml", "--debug"},
			want: []string{"--from", "3", "--debug", "workflow.yaml"},
		},
		{
			name: "负数值是 flag 的值_不被误判为 flag",
			args: []string{"workflow.yaml", "--from", "-3"},
			want: []string{"--from", "-3", "workflow.yaml"},
		},
		{
			name: "单破折号视为位置参数",
			args: []string{"-", "--debug"},
			want: []string{"--debug", "-"},
		},
		{
			name: "双破折号终止符_之后全部视为位置参数",
			args: []string{"--", "workflow.yaml", "--debug"},
			want: []string{"--", "workflow.yaml", "--debug"},
		},
		{
			name: "终止符之前的 flag 仍被前移",
			args: []string{"--debug", "--", "workflow.yaml"},
			want: []string{"--debug", "--", "workflow.yaml"},
		},
		{
			name: "单破折号形式的 flag 同样支持",
			args: []string{"workflow.yaml", "-from", "2"},
			want: []string{"-from", "2", "workflow.yaml"},
		},
		{
			name: "无位置参数时保持原样",
			args: []string{"--out", "workflow.yaml", "--debug"},
			want: []string{"--out", "workflow.yaml", "--debug"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, _, _, _ := newRunLikeFlagSet("test")
			got := reorderFlags(fs, tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("reorderFlags(%q) =\n  %q\nwant\n  %q", tt.args, got, tt.want)
			}
		})
	}
}

// TestReorderFlags_ThenParse 端到端验证：README 记载的写法
// `run workflow.yaml --from 3 --out dir --debug` 经重排后能被标准 flag 包正确解析。
// 修复前该写法中 flags 会被静默忽略（--from 保持默认值 1）。
func TestReorderFlags_ThenParse(t *testing.T) {
	fs, from, out, debug := newRunLikeFlagSet("test")
	args := reorderFlags(fs, []string{"workflow.yaml", "--from", "3", "--out", "shots", "--debug"})

	if err := fs.Parse(args); err != nil {
		t.Fatalf("fs.Parse 失败: %v", err)
	}
	if *from != 3 {
		t.Errorf("--from = %d, want 3（README 写法未生效）", *from)
	}
	if *out != "shots" {
		t.Errorf("--out = %q, want %q", *out, "shots")
	}
	if !*debug {
		t.Errorf("--debug = false, want true")
	}
	if fs.NArg() != 1 || fs.Arg(0) != "workflow.yaml" {
		t.Errorf("位置参数 = %q, want [workflow.yaml]", fs.Args())
	}
}
