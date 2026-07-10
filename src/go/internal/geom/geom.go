// Package geom 定义屏幕几何相关的共享类型（Point）
package geom

import "fmt"

// Point 屏幕坐标点
type Point struct {
	X, Y int
}

// String 实现 fmt.Stringer 接口，返回点的可读表示
func (p Point) String() string {
	return fmt.Sprintf("(%d, %d)", p.X, p.Y)
}
