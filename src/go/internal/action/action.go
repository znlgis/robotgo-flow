package action

import "robotgo-flow/internal/geom"

// Runner 所有操作实现此接口
type Runner interface {
	Execute(eng Engine) error
}

// Engine action 依赖的引擎接口（由 engine 包实现）。
// 所有方法都是 RPA 原语：鼠标/键盘模拟、图像匹配、浏览器控制、截图。
type Engine interface {
	// Click 鼠标左键单击指定坐标。
	Click(x, y int) error

	// DoubleClick 鼠标左键双击指定坐标。
	DoubleClick(x, y int) error

	// RightClick 鼠标右键点击指定坐标。
	RightClick(x, y int) error

	// Drag 从 (fromX,fromY) 拖拽到 (toX,toY)。
	Drag(fromX, fromY, toX, toY int) error

	// TypeText 在当前焦点输入框输入文本。
	TypeText(text string) error

	// PressKey 按下并释放单个按键（如 "enter", "tab"）。
	PressKey(key string) error

	// PressCombo 按下组合键，最后一个参数为键，前面为修饰键（如 PressCombo("ctrl","a")）。
	PressCombo(keys ...string) error

	// FindElement 在屏幕上查找模板图片，返回模板中心的屏幕坐标。
	FindElement(templatePath string) (geom.Point, error)

	// WaitForElement 轮询等待模板出现，超时返回错误。
	WaitForElement(templatePath string, timeoutSec int) error

	// WaitForElementGone 轮询等待模板消失，超时返回错误。
	WaitForElementGone(templatePath string, timeoutSec int) error

	// ScrollDown 向下滚动指定量。
	ScrollDown(amount int) error

	// ScrollUp 向上滚动指定量。
	ScrollUp(amount int) error

	// RefreshPage 刷新当前页面（Ctrl+R）。
	RefreshPage() error

	// Back 浏览器后退（Alt+Left）。
	Back() error

	// Forward 浏览器前进（Alt+Right）。
	Forward() error

	// SwitchTab 切换到指定编号的浏览器标签页（1-9，Ctrl+数字）。
	SwitchTab(index int) error

	// OpenURL 在浏览器中打开 URL，发后即忘（不返回错误，不阻塞流程）。
	OpenURL(url string)

	// FindBrowserWindow 查找并激活浏览器窗口，记录其屏幕坐标供后续窗口内搜索使用。
	FindBrowserWindow() error

	// CaptureScreen 截取全屏并保存到指定文件名，返回保存路径。
	CaptureScreen(filename string) string

	// Wait 暂停执行 ms 毫秒，在人性化模式下会模拟空闲行为。
	Wait(ms int)
}
