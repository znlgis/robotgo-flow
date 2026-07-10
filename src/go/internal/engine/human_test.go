package engine

import (
	"testing"

	"robotgo-flow/internal/geom"
)

// ---------------------------------------------------------------------------
// 钳制（浮点）
// ---------------------------------------------------------------------------

func TestClamp(t *testing.T) {
	tests := []struct {
		name      string
		v, lo, hi float64
		want      float64
	}{
		{name: "低于范围 → 下限", v: -5, lo: 0, hi: 10, want: 0},
		{name: "高于范围 → 上限", v: 15, lo: 0, hi: 10, want: 10},
		{name: "范围内 → 不变", v: 5, lo: 0, hi: 10, want: 5},
		{name: "精确等于下边界", v: 0, lo: 0, hi: 10, want: 0},
		{name: "精确等于上边界", v: 10, lo: 0, hi: 10, want: 10},
		{name: "负范围，范围内", v: -7, lo: -10, hi: -1, want: -7},
		{name: "负范围，低于下限", v: -15, lo: -10, hi: -1, want: -10},
		{name: "负范围，高于上限", v: 0, lo: -10, hi: -1, want: -1},
		{name: "零宽度范围，v 低于（夹到等高值）", v: 3, lo: 7, hi: 7, want: 7},
		{name: "零宽度范围，v 高于", v: 9, lo: 7, hi: 7, want: 7},
		{name: "v 恰好在高低值之间", v: 5, lo: 0, hi: 10, want: 5},
		{name: "low > high：v < low 时返回 low", v: 5, lo: 10, hi: 0, want: 10},
		{name: "low > high，v 在两者之间：返回 high", v: 7, lo: 5, hi: 2, want: 2},
		{name: "low > high，v 高于 high 但低于 low", v: 3, lo: 10, hi: 0, want: 10},
		{name: "小数值", v: 3.14, lo: 0, hi: 5, want: 3.14},
		{name: "小数边界", v: 3.14, lo: 3.14, hi: 9.9, want: 3.14},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clamp(tt.v, tt.lo, tt.hi)
			if got != tt.want {
				t.Errorf("clamp(%v, %v, %v) = %v, want %v", tt.v, tt.lo, tt.hi, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 钳制（整数）
// ---------------------------------------------------------------------------

func TestClampInt(t *testing.T) {
	tests := []struct {
		name      string
		v, lo, hi int
		want      int
	}{
		{name: "低于范围 → 下限", v: -5, lo: 0, hi: 10, want: 0},
		{name: "高于范围 → 上限", v: 15, lo: 0, hi: 10, want: 10},
		{name: "范围内 → 不变", v: 5, lo: 0, hi: 10, want: 5},
		{name: "精确等于下边界", v: 0, lo: 0, hi: 10, want: 0},
		{name: "精确等于上边界", v: 10, lo: 0, hi: 10, want: 10},
		{name: "负范围，范围内", v: -7, lo: -10, hi: -1, want: -7},
		{name: "负范围，低于下限", v: -15, lo: -10, hi: -1, want: -10},
		{name: "负范围，高于上限", v: 0, lo: -10, hi: -1, want: -1},
		{name: "零宽度范围，v 低于", v: 3, lo: 7, hi: 7, want: 7},
		{name: "零宽度范围，v 高于", v: 9, lo: 7, hi: 7, want: 7},
		{name: "中点", v: 5, lo: 0, hi: 10, want: 5},
		{name: "low > high，v 低于 low", v: 2, lo: 10, hi: 3, want: 10},
		{name: "low > high，v 在两者之间", v: 5, lo: 10, hi: 3, want: 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampInt(tt.v, tt.lo, tt.hi)
			if got != tt.want {
				t.Errorf("clampInt(%d, %d, %d) = %d, want %d", tt.v, tt.lo, tt.hi, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 贝塞尔插值点
// ---------------------------------------------------------------------------

func TestBezierPoint_StartPoint(t *testing.T) {
	// t=0 时返回起点 p0，与控制点无关。
	p0 := geom.Point{X: 10, Y: 20}
	p1 := geom.Point{X: 99, Y: 88}
	p2 := geom.Point{X: 77, Y: 66}
	p3 := geom.Point{X: 55, Y: 44}

	got := bezierPoint(p0, p1, p2, p3, 0)
	if got != p0 {
		t.Errorf("bezierPoint(..., t=0) = %v, want %v", got, p0)
	}
}

func TestBezierPoint_EndPoint(t *testing.T) {
	// t=1 时返回终点 p3，与控制点无关。
	p0 := geom.Point{X: 10, Y: 20}
	p1 := geom.Point{X: 99, Y: 88}
	p2 := geom.Point{X: 77, Y: 66}
	p3 := geom.Point{X: 55, Y: 44}

	got := bezierPoint(p0, p1, p2, p3, 1)
	if got != p3 {
		t.Errorf("bezierPoint(..., t=1) = %v, want %v", got, p3)
	}
}

func TestBezierPoint_CollinearMidpoint(t *testing.T) {
	// 四点共线且 p1、p2 关于中点对称时（即 p1.X+p2.X == p0.X+p3.X），
	// t=0.5 处的曲线点应恰好落在 p0 与 p3 的中点。
	p0 := geom.Point{X: 0, Y: 0}
	p1 := geom.Point{X: 30, Y: 30}
	p2 := geom.Point{X: 70, Y: 70}
	p3 := geom.Point{X: 100, Y: 100}

	got := bezierPoint(p0, p1, p2, p3, 0.5)
	want := geom.Point{X: 50, Y: 50}
	if got != want {
		t.Errorf("bezierPoint(diagonal collinear, t=0.5) = %v, want %v", got, want)
	}
}

func TestBezierPoint_CollinearHorizontalMidpoint(t *testing.T) {
	// 同上的共线性质，验证严格水平线。
	p0 := geom.Point{X: 0, Y: 42}
	p1 := geom.Point{X: 25, Y: 42}
	p2 := geom.Point{X: 75, Y: 42}
	p3 := geom.Point{X: 100, Y: 42}

	got := bezierPoint(p0, p1, p2, p3, 0.5)
	want := geom.Point{X: 50, Y: 42}
	if got != want {
		t.Errorf("bezierPoint(horizontal collinear, t=0.5) = %v, want %v", got, want)
	}
}

func TestBezierPoint_KnownValues(t *testing.T) {
	// 通用曲线的预计算值。
	// 控制点: p0=(10,10), p1=(60,80), p2=(140,20), p3=(190,90)
	p0 := geom.Point{X: 10, Y: 10}
	p1 := geom.Point{X: 60, Y: 80}
	p2 := geom.Point{X: 140, Y: 20}
	p3 := geom.Point{X: 190, Y: 90}

	tests := []struct {
		t    float64
		want geom.Point
	}{
		{t: 0.0, want: geom.Point{X: 10, Y: 10}},
		{t: 0.25, want: geom.Point{X: 52, Y: 42}},
		{t: 0.5, want: geom.Point{X: 100, Y: 50}},
		{t: 0.75, want: geom.Point{X: 148, Y: 58}},
		{t: 1.0, want: geom.Point{X: 190, Y: 90}},
	}

	for _, tt := range tests {
		got := bezierPoint(p0, p1, p2, p3, tt.t)
		if got != tt.want {
			t.Errorf("bezierPoint(..., t=%.2f) = %v, want %v", tt.t, got, tt.want)
		}
	}
}

func TestBezierPoint_StraightLineUniformSpacing(t *testing.T) {
	// 控制点沿直线均匀分布时，参数化为线性 —— t=0.25 应对应线段的 25% 处。
	p0 := geom.Point{X: 0, Y: 0}
	p1 := geom.Point{X: 33, Y: 0}
	p2 := geom.Point{X: 67, Y: 0}
	p3 := geom.Point{X: 100, Y: 0}

	got := bezierPoint(p0, p1, p2, p3, 0.25)
	// 均匀间距 (p0=0, p1=33, p2=67, p3=100):
	// 0.421875*0 + 3*0.5625*0.25*33 + 3*0.75*0.0625*67 + 0.015625*100
	// = 0 + 13.921875 + 9.421875 + 1.5625 = 24.90625 → 25
	want := geom.Point{X: 25, Y: 0}
	if got != want {
		t.Errorf("bezierPoint(uniform straight line, t=0.25) = %v, want %v", got, want)
	}
}

// ---------------------------------------------------------------------------
// 基准测试
// ---------------------------------------------------------------------------

func BenchmarkBezierPoint(b *testing.B) {
	p0 := geom.Point{X: 10, Y: 10}
	p1 := geom.Point{X: 60, Y: 80}
	p2 := geom.Point{X: 140, Y: 20}
	p3 := geom.Point{X: 190, Y: 90}

	for b.Loop() {
		_ = bezierPoint(p0, p1, p2, p3, 0.5)
	}
}

func BenchmarkClamp(b *testing.B) {
	for b.Loop() {
		clamp(50, 0, 100)
	}
}

func BenchmarkClampInt(b *testing.B) {
	for b.Loop() {
		clampInt(50, 0, 100)
	}
}
