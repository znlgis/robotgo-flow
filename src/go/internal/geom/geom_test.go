package geom

import (
	"testing"
)

func TestPointEquality(t *testing.T) {
	p1 := Point{X: 10, Y: 20}
	p2 := Point{X: 10, Y: 20}
	p3 := Point{X: 10, Y: 21}

	if p1 != p2 {
		t.Error("相同的点应相等")
	}
	if p1 == p3 {
		t.Error("不同的点不应相等")
	}
}
