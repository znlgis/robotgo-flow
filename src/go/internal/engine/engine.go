package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	bitmaputil "github.com/vcaesar/bitmap"

	"github.com/go-vgo/robotgo"

	"robotgo-flow/internal/action"
	"robotgo-flow/internal/encoding"
	"robotgo-flow/internal/geom"
	"robotgo-flow/internal/logger"
)

// 编译期检查：Engine 必须实现 action.Engine 接口
var _ action.Engine = (*Engine)(nil)

// Engine 是 RPA 自动化引擎，封装了 robotgo 的核心操作
type Engine struct {
	// 模板图片目录，用于图像匹配定位 UI 元素
	templateDir string
	// 截图保存目录，用于调试
	screenshotDir string
	// 默认等待间隔（毫秒）
	defaultDelay int
	// human 人类行为模拟模式（nil = 关闭）
	human *humanMode
	// 浏览器窗口信息（方案 A+B：自动定位 + 窗口内搜索）
	browserPID       int
	windowX, windowY int
	windowW, windowH int
	// 浏览器操作等待延迟（毫秒）
	browserRefreshDelay    int
	browserNavigationDelay int
	browserPageLoadDelay   int
	// 模板图片缓存，避免重复打开
	templateCache   map[string]robotgo.CBitmap
	templateSizes   map[string]geom.Point // 模板尺寸缓存（W/H），避免重复读取文件
	templateCacheMu sync.RWMutex
}

// NewEngine 创建一个新的自动化引擎
func NewEngine(templateDir, screenshotDir string) *Engine {
	os.MkdirAll(screenshotDir, 0755)
	return &Engine{
		templateDir:   templateDir,
		screenshotDir: screenshotDir,
		defaultDelay:  500,
		// 默认浏览器操作延迟（毫秒）
		browserRefreshDelay:    3000,
		browserNavigationDelay: 2000,
		browserPageLoadDelay:   3000,
		// 模板位图缓存
		templateCache: make(map[string]robotgo.CBitmap),
		templateSizes: make(map[string]geom.Point),
	}
}

// EnableHuman 启用人类行为模拟模式
func (e *Engine) EnableHuman(enabled bool, speed, mistakeRate float64, idle, scroll, overshoot bool) {
	e.human = newHumanMode(enabled, speed, mistakeRate, idle, scroll, overshoot)
	if e.human != nil {
		logger.Info("[Engine] 人类行为模拟已启用 (speed=%.1f, mistakeRate=%.2f)", speed, mistakeRate)
	}
}

// ---------------------------------------------------------------------------
// 鼠标操作
// ---------------------------------------------------------------------------

// Click 在指定坐标点击
func (e *Engine) Click(x, y int) error {
	e.moveTo(x, y)
	if err := robotgo.Click(); err != nil {
		return fmt.Errorf("单击失败: %w", err)
	}
	logger.Info("[Click] (%d, %d)", x, y)
	e.delay()
	return nil
}

// DoubleClick 在指定坐标双击
func (e *Engine) DoubleClick(x, y int) error {
	e.moveTo(x, y)
	if err := robotgo.Click(); err != nil {
		return fmt.Errorf("双击(第1次)失败: %w", err)
	}
	if e.human != nil {
		robotgo.MilliSleep(e.human.randomDelay(60, 200))
	} else {
		robotgo.MilliSleep(100)
	}
	if err := robotgo.Click(); err != nil {
		return fmt.Errorf("双击(第2次)失败: %w", err)
	}
	logger.Info("[DoubleClick] (%d, %d)", x, y)
	e.delay()
	return nil
}

// RightClick 在指定坐标右键点击
func (e *Engine) RightClick(x, y int) error {
	e.moveTo(x, y)
	if err := robotgo.Click("right"); err != nil {
		return fmt.Errorf("右键单击失败: %w", err)
	}
	logger.Info("[RightClick] (%d, %d)", x, y)
	e.delay()
	return nil
}

// Drag 执行鼠标拖拽，从 (fromX,fromY) 拖到 (toX,toY)
// 使用 defer 保证 MouseUp 始终被调用，避免鼠标"按下"状态泄漏
func (e *Engine) Drag(fromX, fromY, toX, toY int) error {
	e.moveTo(fromX, fromY)
	if err := robotgo.MouseDown(); err != nil {
		return fmt.Errorf("拖拽按下失败: %w", err)
	}
	defer func() {
		if upErr := robotgo.MouseUp(); upErr != nil {
			logger.DefaultLogger.Error("拖拽释放失败: %v", upErr)
		}
	}()
	e.moveTo(toX, toY)
	logger.Info("[Drag] (%d,%d) -> (%d,%d)", fromX, fromY, toX, toY)
	e.delay()
	return nil
}

// ---------------------------------------------------------------------------
// 键盘操作
// ---------------------------------------------------------------------------

// TypeText 输入文本（适用于输入框已获得焦点的情况）
func (e *Engine) TypeText(text string) error {
	if e.human != nil {
		e.human.typeText(text)
	} else {
		robotgo.Type(text)
	}
	logger.Info("[TypeText] %q", text)
	e.delay()
	return nil
}

// PressKey 按下指定按键
func (e *Engine) PressKey(key string) error {
	if e.human != nil {
		robotgo.MilliSleep(e.human.randomDelay(20, 80))
	}
	if err := robotgo.KeyTap(key); err != nil {
		return fmt.Errorf("按键 %s 失败: %w", key, err)
	}
	logger.Info("[PressKey] %s", key)
	e.delay()
	return nil
}

// PressCombo 按下组合键，如 "ctrl", "a"
func (e *Engine) PressCombo(keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	if e.human != nil {
		robotgo.MilliSleep(e.human.randomDelay(40, 150))
	}
	key := keys[len(keys)-1]
	modifiers := keys[:len(keys)-1]
	var err error
	if len(modifiers) == 0 {
		err = robotgo.KeyTap(key)
	} else {
		args := make([]interface{}, len(modifiers))
		for i, m := range modifiers {
			args[i] = m
		}
		err = robotgo.KeyTap(key, args...)
	}
	if err != nil {
		return fmt.Errorf("组合键 %v 失败: %w", keys, err)
	}
	logger.Info("[PressCombo] %v", keys)
	e.delay()
	return nil
}

// Enter 按下回车键
func (e *Engine) Enter() error {
	return e.PressKey("enter")
}

// ---------------------------------------------------------------------------
// 滚动操作
// ---------------------------------------------------------------------------

// ScrollDown 向下滚动
func (e *Engine) ScrollDown(amount int) error {
	if e.human != nil {
		e.human.scrollStepped(amount, "down")
	} else {
		robotgo.ScrollDir(amount, "down")
	}
	logger.Info("[ScrollDown] amount=%d", amount)
	e.delay()
	return nil
}

// ScrollUp 向上滚动
func (e *Engine) ScrollUp(amount int) error {
	if e.human != nil {
		e.human.scrollStepped(amount, "up")
	} else {
		robotgo.ScrollDir(amount, "up")
	}
	logger.Info("[ScrollUp] amount=%d", amount)
	e.delay()
	return nil
}

// ---------------------------------------------------------------------------
// 等待与超时
// ---------------------------------------------------------------------------

// Wait 等待指定毫秒数
func (e *Engine) Wait(ms int) {
	if e.human != nil && e.human.idleBehavior {
		e.human.idleWait(ms)
	} else {
		robotgo.MilliSleep(ms)
	}
}

// ---------------------------------------------------------------------------
// 剪贴板操作
// ---------------------------------------------------------------------------

// CopyToClipboard 将文本复制到剪贴板
func (e *Engine) CopyToClipboard(text string) {
	if err := robotgo.WriteAll(text); err != nil {
		logger.Info("[CopyToClipboard] 写入剪贴板失败: %v", err)
	}
	logger.Info("[CopyToClipboard] %q", text)
}

// PasteFromClipboard 从剪贴板粘贴
func (e *Engine) PasteFromClipboard() {
	_ = e.PressCombo("ctrl", "v")
	logger.Info("[PasteFromClipboard]")
	e.delay()
}

// ---------------------------------------------------------------------------
// 截图与调试
// ---------------------------------------------------------------------------

// CaptureScreen 截取全屏并保存
func (e *Engine) CaptureScreen(filename string) string {
	bitmap := robotgo.CaptureScreen()
	if bitmap == nil {
		logger.Info("[CaptureScreen] 截图失败")
		return ""
	}
	defer robotgo.FreeBitmap(bitmap)

	savePath := filepath.Join(e.screenshotDir, filename)
	if err := bitmaputil.Save(bitmap, encoding.ToGBK(savePath)); err != nil {
		logger.Info("[CaptureScreen] 保存截图失败: %v", err)
		return ""
	}
	logger.Info("[CaptureScreen] 已保存: %s", savePath)
	return savePath
}

// ---------------------------------------------------------------------------
// 浏览器辅助操作
// ---------------------------------------------------------------------------

// FindBrowserWindow 查找并定位浏览器窗口。
// 注意：此方法非线程安全，不应并发调用。
// 支持 Chrome / Edge / Firefox / Brave / Opera
// 跳过尺寸为 0 的窗口（多进程架构下的后台进程）
func (e *Engine) FindBrowserWindow() error {
	browsers := []string{"chrome.exe", "msedge.exe", "firefox.exe", "brave.exe", "opera.exe"}
	for _, name := range browsers {
		pids, err := robotgo.FindIds(name)
		if err != nil || len(pids) == 0 {
			continue
		}
		for _, pid := range pids {
			// 获取窗口尺寸，跳过无效窗口（后台进程往往返回 0x0）
			x, y, w, h := robotgo.GetBounds(pid)
			if w == 0 || h == 0 {
				continue
			}
			// 跳过过小的窗口（WebView2 嵌入控件等，通常 < 400x300）
			// 这些窗口不是真正的浏览器窗口，ActivePid/MaxWindow 会导致卡死
			if w < 400 || h < 300 {
				continue
			}

			// 激活窗口
			if err := robotgo.ActivePid(pid); err != nil {
				continue
			}
			robotgo.MilliSleep(500)

			// 最大化
			robotgo.MaxWindow(pid)
			robotgo.MilliSleep(500)

			// 重新获取最大化后的尺寸
			x, y, w, h = robotgo.GetBounds(pid)

			e.browserPID = pid
			e.windowX = x
			e.windowY = y
			e.windowW = w
			e.windowH = h

			logger.Info("[Browser] 已定位浏览器: %s (PID=%d) 窗口=(%d,%d) %dx%d", name, pid, x, y, w, h)
			return nil
		}
	}
	return fmt.Errorf("未找到浏览器窗口")
}

// OpenURL 在浏览器中打开 URL（通过地址栏），加载完成后自动定位浏览器窗口
// 注意：会模拟 Ctrl+L / Ctrl+V / Enter 按键，确保浏览器窗口为当前激活窗口
func (e *Engine) OpenURL(url string) {
	// 优先激活浏览器窗口，避免在非浏览器窗口（如终端）中执行危险操作
	if err := e.FindBrowserWindow(); err != nil {
		logger.Info("[Browser] 警告: %v — URL 操作可能发送到当前激活窗口", err)
	}
	e.CopyToClipboard(url)
	if err := e.PressCombo("ctrl", "l"); err != nil {
		logger.Info("[OpenURL] 聚焦地址栏失败: %v", err)
	}
	e.PasteFromClipboard()
	robotgo.MilliSleep(300)
	if err := e.Enter(); err != nil {
		logger.Info("[OpenURL] 按回车失败: %v", err)
	}
	logger.Info("[OpenURL] %s", url)
	e.Wait(e.browserPageLoadDelay) // 等待页面加载

	// 方案 A：重新定位浏览器窗口（页面加载后窗口属性可能变化）
	if err := e.FindBrowserWindow(); err != nil {
		logger.Info("[Browser] 警告: %v (将继续使用全屏搜索)", err)
	}
}

// RefreshPage 刷新当前页面
func (e *Engine) RefreshPage() error {
	if err := e.PressCombo("ctrl", "r"); err != nil {
		return err
	}
	robotgo.MilliSleep(e.browserRefreshDelay)
	return nil
}

// Back 浏览器返回上一页
func (e *Engine) Back() error {
	if err := e.PressCombo("alt", "left"); err != nil {
		return err
	}
	robotgo.MilliSleep(e.browserNavigationDelay)
	return nil
}

// Forward 浏览器前进下一页
func (e *Engine) Forward() error {
	if err := e.PressCombo("alt", "right"); err != nil {
		return err
	}
	robotgo.MilliSleep(e.browserNavigationDelay)
	return nil
}

// SwitchTab 切换到指定编号的浏览器标签页 (1-9)
func (e *Engine) SwitchTab(index int) error {
	if index < 1 || index > 9 {
		return fmt.Errorf("切换标签: 序号必须为 1-9")
	}
	key := fmt.Sprintf("%d", index)
	return e.PressCombo("ctrl", key)
}

// ---------------------------------------------------------------------------
// 内部辅助方法
// ---------------------------------------------------------------------------

func (e *Engine) moveTo(x, y int) {
	if e.human != nil {
		e.human.moveMouseTo(x, y)
		robotgo.MilliSleep(e.human.clickDelay())
	} else {
		robotgo.MoveSmooth(x, y)
		robotgo.MilliSleep(100)
	}
}

func (e *Engine) delay() {
	if e.human != nil {
		ms := e.human.randomDelay(200, 800)
		robotgo.MilliSleep(ms)
	} else {
		robotgo.MilliSleep(e.defaultDelay)
	}
}

// Close 释放引擎持有的资源（包括缓存的模板位图）
func (e *Engine) Close() {
	e.templateCacheMu.Lock()
	defer e.templateCacheMu.Unlock()
	for path, bitmap := range e.templateCache {
		robotgo.FreeBitmap(bitmap)
		delete(e.templateCache, path)
	}
	clear(e.templateSizes)
}

// openCachedTemplate 打开模板位图，优先从缓存中获取。
// 参数 path 应为 UTF-8 编码的 clean path，内部自动转换为 GBK 路径供 CGo 使用。
// 调用方使用返回的位图时不应释放它，缓存由 Engine.Close 统一管理。
func (e *Engine) openCachedTemplate(path string) robotgo.CBitmap {
	e.templateCacheMu.RLock()
	cached, ok := e.templateCache[path]
	e.templateCacheMu.RUnlock()
	if ok {
		return cached
	}

	sysPath := encoding.ToGBK(path)
	bitmap := bitmaputil.Open(sysPath)
	if bitmap == nil {
		return nil
	}

	// 双重检查：并发调用时可能已有其它 goroutine 写入缓存，
	// 此时释放本次新打开的位图，避免内存泄漏与缓存被覆盖后无人释放。
	e.templateCacheMu.Lock()
	if existing, ok := e.templateCache[path]; ok {
		e.templateCacheMu.Unlock()
		robotgo.FreeBitmap(bitmap)
		return existing
	}
	e.templateCache[path] = bitmap
	e.templateCacheMu.Unlock()
	return bitmap
}

// templateSize 返回模板图片的宽高，结果缓存在内存中避免每次匹配都读取文件。
func (e *Engine) templateSize(path string) (int, int, error) {
	e.templateCacheMu.RLock()
	size, ok := e.templateSizes[path]
	e.templateCacheMu.RUnlock()
	if ok {
		return size.X, size.Y, nil
	}

	w, h, err := robotgo.ImgSize(encoding.ToGBK(path))
	if err != nil {
		return 0, 0, fmt.Errorf("查找元素: 获取图片尺寸失败: %w", err)
	}

	e.templateCacheMu.Lock()
	e.templateSizes[path] = geom.Point{X: w, Y: h}
	e.templateCacheMu.Unlock()
	return w, h, nil
}
