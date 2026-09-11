package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-vgo/robotgo"
	bitmaputil "github.com/vcaesar/bitmap"

	"robotgo-flow/internal/geom"
)

// 模板相关的哨兵错误：调用方（如 WaitForElementGone）需要区分
// “模板文件不存在/不可读”与“屏幕上暂时找不到”，前者属于配置错误，应立即失败而不是轮询等待。
var (
	// ErrTemplateNotFound 表示模板文件不存在。
	ErrTemplateNotFound = errors.New("模板文件不存在")
	// ErrTemplateUnreadable 表示模板文件无法解码为位图。
	ErrTemplateUnreadable = errors.New("模板文件无法读取")
)

// IsTemplateError 报告 err 是否由模板文件本身缺失/不可读导致。
func IsTemplateError(err error) bool {
	return errors.Is(err, ErrTemplateNotFound) || errors.Is(err, ErrTemplateUnreadable)
}

// ---------------------------------------------------------------------------
// 图像匹配定位
// ---------------------------------------------------------------------------

// resolveTemplate 将配置中的模板名解析为绝对路径并校验文件存在。
// 绝对路径原样使用（兼容 YAML 中的绝对模板路径），相对路径以 templateDir 为基准。
func (e *Engine) resolveTemplate(templateName string) (string, error) {
	if templateName == "" {
		return "", fmt.Errorf("查找元素: %w (模板名为空)", ErrTemplateNotFound)
	}
	path := templateName
	if !filepath.IsAbs(path) {
		path = filepath.Join(e.templateDir, templateName)
	}
	cleanPath := filepath.Clean(path)

	if _, err := os.Stat(cleanPath); err != nil {
		// 使用 UTF-8 绝对路径报错，便于用户定位问题
		if abs, absErr := filepath.Abs(cleanPath); absErr == nil {
			cleanPath = abs
		}
		return "", fmt.Errorf("查找元素: %w: %s", ErrTemplateNotFound, cleanPath)
	}
	return cleanPath, nil
}

// FindElement 在屏幕上查找模板图片，返回模板中心的屏幕坐标。
// 当已定位浏览器窗口时，优先在窗口内搜索以避免匹配浏览器外的无关元素。
func (e *Engine) FindElement(templateName string) (geom.Point, error) {
	cleanPath, err := e.resolveTemplate(templateName)
	if err != nil {
		return geom.Point{}, err
	}

	// 方案 B：优先在浏览器窗口内搜索
	if x, y, ok := e.findInWindow(cleanPath); ok {
		return e.centerOf(cleanPath, x, y)
	}

	// 回退：全屏搜索（openCachedTemplate 内部做 GBK 转换）
	bitmap := e.openCachedTemplate(cleanPath)
	if bitmap == nil {
		return geom.Point{}, fmt.Errorf("查找元素: %w: %s", ErrTemplateUnreadable, cleanPath)
	}

	x, y := bitmaputil.Find(bitmap)
	if x < 0 || y < 0 {
		return geom.Point{}, fmt.Errorf("查找元素: 在屏幕上未找到 %s (请确认截图与当前 UI 一致、分辨率匹配)", cleanPath)
	}
	return e.centerOf(cleanPath, x, y)
}

// centerOf 将模板左上角坐标转换为模板中心坐标。
func (e *Engine) centerOf(templatePath string, x, y int) (geom.Point, error) {
	w, h, err := e.templateSize(templatePath)
	if err != nil {
		return geom.Point{}, err
	}
	return geom.Point{X: x + w/2, Y: y + h/2}, nil
}

// ---------------------------------------------------------------------------
// 等待与超时
// ---------------------------------------------------------------------------

// waitCondition 通用轮询条件等待。
// check 返回 (满足, err)：err 非 nil 时立即中断并返回该错误，避免无谓地等待到超时。
// 超时未满足时返回 timeoutErr。
func (e *Engine) waitCondition(timeout, interval time.Duration, timeoutErr error, check func() (bool, error)) error {
	deadline := time.Now().Add(timeout)
	for {
		ok, err := check()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		if !time.Now().Before(deadline) {
			return timeoutErr
		}
		time.Sleep(interval)
	}
}

// WaitForElement 等待 UI 元素出现。
func (e *Engine) WaitForElement(templateName string, timeoutSec int) error {
	// 模板文件本身缺失时不必等待：这是配置错误，立即失败
	if _, err := e.resolveTemplate(templateName); err != nil {
		return err
	}
	return e.waitCondition(
		time.Duration(timeoutSec)*time.Second,
		200*time.Millisecond,
		fmt.Errorf("等待元素: %d 秒后超时: %s", timeoutSec, templateName),
		func() (bool, error) {
			_, err := e.FindElement(templateName)
			if err == nil {
				return true, nil
			}
			if IsTemplateError(err) {
				return false, err // 模板缺失/不可读：立即失败
			}
			return false, nil // 尚未出现，继续轮询
		},
	)
}

// WaitForElementGone 等待 UI 元素消失。
func (e *Engine) WaitForElementGone(templateName string, timeoutSec int) error {
	// 模板文件本身缺失时无法判断“是否消失”，立即失败而不是误判为已消失
	if _, err := e.resolveTemplate(templateName); err != nil {
		return err
	}
	return e.waitCondition(
		time.Duration(timeoutSec)*time.Second,
		200*time.Millisecond,
		fmt.Errorf("等待元素消失: %d 秒后仍可见: %s", timeoutSec, templateName),
		func() (bool, error) {
			_, err := e.FindElement(templateName)
			if err == nil {
				return false, nil // 仍可见
			}
			if IsTemplateError(err) {
				return false, err
			}
			return true, nil // 已消失
		},
	)
}

// ---------------------------------------------------------------------------
// 窗口内区域搜索（方案 B：避免匹配到浏览器外的无关元素）
// ---------------------------------------------------------------------------

// findInWindow 在浏览器窗口区域内搜索模板。
// 若找不到或浏览器窗口未知，返回 false；调用方应回退到全屏搜索。
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
	return e.windowX + x, e.windowY + y, true
}
