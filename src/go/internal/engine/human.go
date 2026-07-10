package engine

import (
	"math"
	"math/rand"
	"time"

	"github.com/go-vgo/robotgo"

	"robotgo-flow/internal/geom"
)

// humanMode 封装所有人类行为模拟逻辑
type humanMode struct {
	speed         float64
	mistakeRate   float64
	idleBehavior  bool
	scrollJitter  bool
	moveOvershoot bool

	// 衍生值，从 speed 计算
	typeDelayMin  int
	typeDelayMax  int
	clickDelayMin int
	clickDelayMax int
}

// newHumanMode 从 config 建立 humanMode 实例
func newHumanMode(enabled bool, speed, mistakeRate float64, idle, scroll, overshoot bool) *humanMode {
	if !enabled {
		return nil
	}
	h := &humanMode{
		speed:         clamp(speed, 0.1, 5.0),
		mistakeRate:   clamp(mistakeRate, 0.0, 1.0),
		idleBehavior:  idle,
		scrollJitter:  scroll,
		moveOvershoot: overshoot,
	}
	// 衍生延迟范围：speed 越大越快（延迟越小）
	h.typeDelayMin = clampInt(int(50.0/h.speed), 10, 500)
	h.typeDelayMax = clampInt(int(250.0/h.speed), 30, 1000)
	if h.typeDelayMax < h.typeDelayMin {
		h.typeDelayMax = h.typeDelayMin + 20
	}
	h.clickDelayMin = clampInt(int(80.0/h.speed), 20, 600)
	h.clickDelayMax = clampInt(int(350.0/h.speed), 60, 1500)
	if h.clickDelayMax < h.clickDelayMin {
		h.clickDelayMax = h.clickDelayMin + 30
	}
	return h
}

// clamp 将值限制在 [low, high] 范围内
func clamp(v, low, high float64) float64 {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

// clampInt 将整数值限制在 [low, high] 范围内
func clampInt(v, low, high int) int {
	if v < low {
		return low
	}
	if v > high {
		return high
	}
	return v
}

// ---------------------------------------------------------------------------
// 随机延迟
// ---------------------------------------------------------------------------

// randomDelay 返回 min 到 max 之间的随机毫秒数
func (h *humanMode) randomDelay(minMs, maxMs int) int {
	if minMs >= maxMs {
		return minMs
	}
	return minMs + rand.Intn(maxMs-minMs)
}

// clickDelay 返回点击前的随机等待毫秒数
func (h *humanMode) clickDelay() int {
	return h.randomDelay(h.clickDelayMin, h.clickDelayMax)
}

// typeDelay 返回两个按键之间的随机等待毫秒数
func (h *humanMode) typeDelay() int {
	return h.randomDelay(h.typeDelayMin, h.typeDelayMax)
}

// ---------------------------------------------------------------------------
// 贝塞尔曲线鼠标路径
// ---------------------------------------------------------------------------

// bezierPoint 计算三次贝塞尔曲线上 t 位置的点
func bezierPoint(p0, p1, p2, p3 geom.Point, t float64) geom.Point {
	mt := 1.0 - t
	mt2 := mt * mt
	mt3 := mt2 * mt
	t2 := t * t
	t3 := t2 * t

	x := mt3*float64(p0.X) + 3*mt2*t*float64(p1.X) + 3*mt*t2*float64(p2.X) + t3*float64(p3.X)
	y := mt3*float64(p0.Y) + 3*mt2*t*float64(p1.Y) + 3*mt*t2*float64(p2.Y) + t3*float64(p3.Y)

	return geom.Point{X: int(math.Round(x)), Y: int(math.Round(y))}
}

// bezierControlPoints 为起点和终点生成随机控制点
func (h *humanMode) bezierControlPoints(from, to geom.Point) (geom.Point, geom.Point) {
	dx := to.X - from.X
	dy := to.Y - from.Y
	dist := math.Sqrt(float64(dx*dx + dy*dy))

	// 控制点偏移量：距离的 20%-50%
	offset := dist * (0.2 + rand.Float64()*0.3)

	// P1 靠近起点，随机偏移
	angle1 := (rand.Float64() - 0.5) * math.Pi / 2 // ±45度
	p1x := float64(from.X) + offset*math.Cos(angle1)
	p1y := float64(from.Y) + offset*math.Sin(angle1)

	// P2 靠近终点，随机偏移
	angle2 := (rand.Float64() - 0.5) * math.Pi / 2
	p2x := float64(to.X) - offset*math.Cos(angle2)
	p2y := float64(to.Y) - offset*math.Sin(angle2)

	return geom.Point{X: int(math.Round(p1x)), Y: int(math.Round(p1y))},
		geom.Point{X: int(math.Round(p2x)), Y: int(math.Round(p2y))}
}

// bezierPath 产生从 from 到 to 的贝塞尔曲线路径（返回中间点序列）
func (h *humanMode) bezierPath(from, to geom.Point) []geom.Point {
	p1, p2 := h.bezierControlPoints(from, to)

	// 根据距离决定采样点数量（每像素约0.8个点，最少50，最多200）
	dx := to.X - from.X
	dy := to.Y - from.Y
	dist := math.Sqrt(float64(dx*dx + dy*dy))
	numPoints := clampInt(int(dist*0.8), 50, 200)

	path := make([]geom.Point, numPoints)
	for i := 0; i < numPoints; i++ {
		t := float64(i) / float64(numPoints-1)
		path[i] = bezierPoint(from, p1, p2, to, t)
	}

	return path
}

// ---------------------------------------------------------------------------
// 路径行走（可变速度）
// ---------------------------------------------------------------------------

// walkPath 沿路径移动鼠标，靠近终点时减速
func (h *humanMode) walkPath(path []geom.Point) {
	n := len(path)
	if n <= 1 {
		if n == 1 {
			// 只有一个点，直接移动
			robotgo.Move(path[0].X, path[0].Y)
			time.Sleep(time.Duration(h.clickDelay()) * time.Millisecond)
		}
		return
	}

	for i, pt := range path {
		robotgo.Move(pt.X, pt.Y)

		// 根据路径进度决定间隔时间（n > 1 时 n-1 不会为零）
		progress := float64(i) / float64(n-1)

		var sleepMs int
		switch {
		case progress < 0.3:
			sleepMs = h.randomDelay(1, 3)
		case progress < 0.7:
			sleepMs = h.randomDelay(2, 5)
		case progress < 0.9:
			sleepMs = h.randomDelay(5, 12)
		default:
			sleepMs = h.randomDelay(10, 25)
		}
		robotgo.MilliSleep(sleepMs)
	}
}

// moveMouseTo 使用人类化路径移动鼠标到目标位置
func (h *humanMode) moveMouseTo(x, y int) {
	curX, curY := robotgo.Location()
	target := geom.Point{X: x, Y: y}
	current := geom.Point{X: curX, Y: curY}

	// 如果移动距离很短（<20px），直接移动
	dx := x - curX
	dy := y - curY
	if math.Sqrt(float64(dx*dx+dy*dy)) < 20 {
		robotgo.Move(x, y)
		robotgo.MilliSleep(h.randomDelay(10, 40))
		return
	}

	// 生成贝塞尔路径并行走
	path := h.bezierPath(current, target)
	h.walkPath(path)

	// 偶尔超过目标再修正
	if h.moveOvershoot && rand.Float64() < 0.15 {
		overshootX := x + rand.Intn(15) - 7
		overshootY := y + rand.Intn(15) - 7
		robotgo.Move(overshootX, overshootY)
		robotgo.MilliSleep(h.randomDelay(30, 80))
		h.walkPath(h.bezierPath(geom.Point{X: overshootX, Y: overshootY}, target))
	}

	// 确保最终位置精确
	robotgo.Move(x, y)
}

// ---------------------------------------------------------------------------
// 人类化打字（变速 + 错误）
// ---------------------------------------------------------------------------

// typeText 以人类化方式输入文字（变速 + 偶尔错误）
func (h *humanMode) typeText(text string) {
	runes := []rune(text)
	i := 0
	for i < len(runes) {
		mistakeType := -1
		if rand.Float64() < h.mistakeRate {
			mistakeType = rand.Intn(3) // 0, 1, or 2
		}

		switch mistakeType {
		case 0:
			// 打错相邻按键，然后退格修正
			h.typeMistakeAdjacent(runes[i])
			i++
		case 1:
			// 跳过当前字符，稍后修正
			if i < len(runes)-1 {
				h.typeCharWithDelay(runes[i+1])
				robotgo.MilliSleep(h.randomDelay(40, 120))
				_ = robotgo.KeyTap("backspace")
				robotgo.MilliSleep(h.randomDelay(40, 100))
				h.typeCharWithDelay(runes[i])
				robotgo.MilliSleep(h.typeDelay())
				h.typeCharWithDelay(runes[i+1])
				i += 2
			} else {
				h.typeCharWithDelay(runes[i])
				i++
			}
		case 2:
			// 交换当前和下一个字符
			if i < len(runes)-1 {
				h.typeCharWithDelay(runes[i+1])
				robotgo.MilliSleep(h.typeDelay())
				h.typeCharWithDelay(runes[i])
				robotgo.MilliSleep(h.randomDelay(60, 200))
				_ = robotgo.KeyTap("backspace")
				robotgo.MilliSleep(h.randomDelay(40, 100))
				_ = robotgo.KeyTap("backspace")
				robotgo.MilliSleep(h.randomDelay(40, 100))
				h.typeCharWithDelay(runes[i])
				robotgo.MilliSleep(h.typeDelay())
				h.typeCharWithDelay(runes[i+1])
				i += 2
			} else {
				h.typeCharWithDelay(runes[i])
				i++
			}
		default:
			// 正常输入
			h.typeCharWithDelay(runes[i])
			i++
		}
	}
}

// typeMistakeAdjacent 模拟打错相邻按键后退格修正
func (h *humanMode) typeMistakeAdjacent(ch rune) {
	adj, ok := adjacentKeys[ch]
	if !ok || len(adj) == 0 {
		h.typeCharWithDelay(ch)
		return
	}

	// 随机选择一个邻近按键
	wrongCh := adj[rand.Intn(len(adj))]

	// 输入错误字符
	robotgo.Type(string(wrongCh))
	robotgo.MilliSleep(h.randomDelay(30, 150))

	// 意识到错误，退格
	_ = robotgo.KeyTap("backspace")
	robotgo.MilliSleep(h.randomDelay(40, 180))

	// 输入正确字符
	h.typeCharWithDelay(ch)
}

// typeCharWithDelay 输入单个字符并等待随机时间
func (h *humanMode) typeCharWithDelay(ch rune) {
	robotgo.Type(string(ch))
	robotgo.MilliSleep(h.typeDelay())
}

// ---------------------------------------------------------------------------
// 滚动
// ---------------------------------------------------------------------------

// scrollStepped 以人类化方式逐步滚动
func (h *humanMode) scrollStepped(amount int, direction string) {
	if !h.scrollJitter || amount <= 1 {
		robotgo.ScrollDir(amount, direction)
		return
	}

	// 将总滚动量拆分为多个小步骤
	remaining := amount
	for remaining > 0 {
		maxStep := remaining
		if maxStep > 3 {
			maxStep = 3
		}
		step := 1 + rand.Intn(maxStep)
		if step > remaining {
			step = remaining
		}
		robotgo.ScrollDir(step, direction)
		remaining -= step
		if remaining > 0 {
			robotgo.MilliSleep(h.randomDelay(30, 120))
		}
	}

	// 偶尔轻微回滚（模拟人类的不精确）
	if rand.Float64() < 0.2 {
		backAmount := 1 + rand.Intn(2)
		robotgo.MilliSleep(h.randomDelay(50, 150))
		backDir := "up"
		if direction == "up" {
			backDir = "down"
		}
		robotgo.ScrollDir(backAmount, backDir)
	}
}

// ---------------------------------------------------------------------------
// 闲置行为
// ---------------------------------------------------------------------------

// idleWait 将长时间等待拆分为多个小段，期间偶尔加入微动作
func (h *humanMode) idleWait(totalMs int) {
	if totalMs <= 0 {
		return
	}
	elapsed := 0
	for elapsed < totalMs {
		chunkMs := h.randomDelay(200, 500)
		if elapsed+chunkMs > totalMs {
			chunkMs = totalMs - elapsed
		}
		robotgo.MilliSleep(chunkMs)
		elapsed += chunkMs

		// 15% 概率在等待期间做微动作
		if h.idleBehavior && rand.Float64() < 0.15 && elapsed < totalMs {
			h.idleWiggle()
		}
	}
}

// idleWiggle 执行一个微小的鼠标抖动
func (h *humanMode) idleWiggle() {
	curX, curY := robotgo.Location()
	offsetX := rand.Intn(10) - 5
	offsetY := rand.Intn(10) - 5

	robotgo.Move(curX+offsetX, curY+offsetY)
	robotgo.MilliSleep(h.randomDelay(30, 100))
	robotgo.Move(curX, curY)
}
