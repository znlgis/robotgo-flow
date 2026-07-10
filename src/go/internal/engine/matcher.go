package engine

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-vgo/robotgo"
	bitmaputil "github.com/vcaesar/bitmap"

	"robotgo-flow/internal/encoding"
	"robotgo-flow/internal/geom"
)

// ---------------------------------------------------------------------------
// 图像匹配定位
// ---------------------------------------------------------------------------

// FindElement 在屏幕上查找模板图片，返回中心坐标
// 当已定位浏览器窗口时，优先在窗口内搜索以避免匹配浏览器外的无关元素
func (e *Engine) FindElement(templateName string) (geom.Point, error) {
	templatePath := filepath.Join(e.templateDir, templateName)
	if _, err := os.Stat(templatePath); os.IsNotExist(err) {
		abs, err := filepath.Abs(templatePath)
		if err != nil {
			abs = templatePath
		}
		return geom.Point{}, fmt.Errorf("查找元素: 模板未找到: %s", abs)
	}

	// 使用 UTF-8 clean path 作为缓存 key，确保路径一致性
	cleanPath := filepath.Clean(templatePath)

	// 方案 B：优先在浏览器窗口内搜索
	if x, y, ok := e.findInWindow(cleanPath); ok {
		sysPath := encoding.ToGBK(cleanPath)
		w, h, err := robotgo.ImgSize(sysPath)
		if err != nil {
			return geom.Point{}, fmt.Errorf("查找元素: 获取图片尺寸失败: %w", err)
		}
		return geom.Point{X: x + w/2, Y: y + h/2}, nil
	}

	// 回退：全屏搜索（openCachedTemplate 内部做 GBK 转换）
	bitmap := e.openCachedTemplate(cleanPath)
	if bitmap == nil {
		return geom.Point{}, fmt.Errorf("查找元素: 无法打开模板: %s", templatePath)
	}

	x, y := bitmaputil.Find(bitmap)
	if x < 0 || y < 0 {
		abs, err := filepath.Abs(templatePath)
		if err != nil {
			abs = templatePath
		}
		return geom.Point{}, fmt.Errorf("查找元素: 在屏幕上未找到 %s (请确认截图与当前 UI 一致、分辨率匹配)", abs)
	}

	sysPath := encoding.ToGBK(cleanPath)
	w, h, err := robotgo.ImgSize(sysPath)
	if err != nil {
		return geom.Point{}, fmt.Errorf("查找元素: 获取图片尺寸失败: %w", err)
	}
	return geom.Point{X: x + w/2, Y: y + h/2}, nil
}

// ---------------------------------------------------------------------------
// 等待与超时
// ---------------------------------------------------------------------------

// waitCondition 通用轮询条件等待
// timeout 最大等待时间，interval 每次检查间隔，check 返回 true 表示条件满足
// 返回值: true 表示条件在超时前满足
func (e *Engine) waitCondition(timeout time.Duration, interval time.Duration, check func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return true
		}
		time.Sleep(interval)
	}
	return false
}

// WaitForElement 等待 UI 元素出现
func (e *Engine) WaitForElement(templateName string, timeoutSec int) error {
	ok := e.waitCondition(time.Duration(timeoutSec)*time.Second, 200*time.Millisecond, func() bool {
		_, err := e.FindElement(templateName)
		return err == nil
	})
	if !ok {
		return fmt.Errorf("等待元素: %d 秒后超时: %s", timeoutSec, templateName)
	}
	return nil
}

// WaitForElementGone 等待 UI 元素消失
func (e *Engine) WaitForElementGone(templateName string, timeoutSec int) error {
	ok := e.waitCondition(time.Duration(timeoutSec)*time.Second, 200*time.Millisecond, func() bool {
		_, err := e.FindElement(templateName)
		return err != nil
	})
	if !ok {
		return fmt.Errorf("等待元素消失: %d 秒后仍可见: %s", timeoutSec, templateName)
	}
	return nil
}

// ---------------------------------------------------------------------------
// 窗口内区域搜索（方案 B：避免匹配到浏览器外的无关元素）
// ---------------------------------------------------------------------------

// findInWindow 在浏览器窗口区域内搜索模板
// 若找不到或浏览器窗口未知，返回 false；调用方应回退到全屏搜索
func (e *Engine) findInWindow(templatePath string) (int, int, bool) {
	if e.browserPID == 0 {
		return 0, 0, false
	}

	// 截取浏览器窗口区域
	region := robotgo.CaptureScreen(e.windowX, e.windowY, e.windowW, e.windowH)
	if region == nil {
		return 0, 0, false
	}
	defer robotgo.FreeBitmap(region)

	bitmap := e.openCachedTemplate(templatePath)
	if bitmap == nil {
		return 0, 0, false
	}

	x, y := bitmaputil.Find(bitmap, region)
	if x < 0 || y < 0 {
		return 0, 0, false
	}

	// 转换为屏幕绝对坐标
	absX := e.windowX + x
	absY := e.windowY + y
	return absX, absY, true
}
