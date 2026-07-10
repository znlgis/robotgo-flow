package capture

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	bitmaputil "github.com/vcaesar/bitmap"

	"github.com/go-vgo/robotgo"

	"robotgo-flow/internal/encoding"
)

// CaptureRegion 截取屏幕指定区域并保存
func CaptureRegion(x, y, w, h int, outputPath string) error {
	bitmap := robotgo.CaptureScreen(x, y, w, h)
	if bitmap == nil {
		return fmt.Errorf("区域截图: 屏幕捕获失败")
	}
	defer robotgo.FreeBitmap(bitmap)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}
	if err := bitmaputil.Save(bitmap, encoding.ToGBK(outputPath)); err != nil {
		return fmt.Errorf("保存截图失败: %w", err)
	}
	return nil
}

// CaptureInteractive 交互式截图：提示用户框选区域
func CaptureInteractive(name, outputDir string) error {
	in := bufio.NewReader(os.Stdin)
	getPoint := func(corner string) (int, int, error) {
		if corner == "左上角" {
			fmt.Printf("请将鼠标移动到 %s 的左上角，按 Enter 确认...\n", name)
		} else {
			fmt.Println("请将鼠标移动到右下角，按 Enter 确认...")
		}
		if _, err := in.ReadString('\n'); err != nil {
			return 0, 0, fmt.Errorf("读取输入失败: %w", err)
		}
		x, y := robotgo.Location()
		label := "起点"
		if corner == "右下角" {
			label = "终点"
		}
		fmt.Printf("%s: (%d, %d)\n", label, x, y)
		return x, y, nil
	}
	return captureRegionInteractive(name, outputDir, getPoint, os.Stdout)
}

// CaptureInteractivePipe 管道模式交互式截图：通过检测 Enter 键替代 stdin 读取。
// 适用于 pipe 模式（stdin 已被 JSON 协议占用）的场景。
// 用户操作：将鼠标移到目标元素左上角按 Enter → 移到右下角按 Enter。
func CaptureInteractivePipe(name, outputDir string) error {
	getPoint := func(corner string) (int, int, error) {
		if corner == "左上角" {
			fmt.Fprintf(os.Stderr, "[截图] 请将鼠标移到 %s 的左上角，按 Enter 确认...\n", name)
		} else {
			fmt.Fprintf(os.Stderr, "请将鼠标移到右下角，按 Enter 确认...\n")
		}
		if err := waitEnterPress(); err != nil {
			return 0, 0, err
		}
		x, y := robotgo.Location()
		label := "起点"
		if corner == "右下角" {
			label = "终点"
		}
		fmt.Fprintf(os.Stderr, "%s: (%d, %d)\n", label, x, y)
		return x, y, nil
	}
	return captureRegionInteractive(name, outputDir, getPoint, os.Stderr)
}

// captureRegionInteractive 交互式截图核心：通过 getPoint 获取两个角点坐标，计算宽高后调用 CaptureRegion
func captureRegionInteractive(name, outputDir string, getPoint func(string) (int, int, error), out io.Writer) error {
	x1, y1, err := getPoint("左上角")
	if err != nil {
		return err
	}
	x2, y2, err := getPoint("右下角")
	if err != nil {
		return err
	}
	w := x2 - x1
	h := y2 - y1
	if w <= 0 || h <= 0 {
		return fmt.Errorf("无效区域: 宽=%d 高=%d", w, h)
	}
	outputPath := filepath.Join(outputDir, name+".png")
	if err := CaptureRegion(x1, y1, w, h, outputPath); err != nil {
		return err
	}
	fmt.Fprintf(out, "模板已保存: %s (%dx%d)\n", outputPath, w, h)
	return nil
}

// waitEnterPress 轮询等待 Enter 键按下后释放，带防抖，默认 60 秒超时
func waitEnterPress() error {
	return waitEnterPressWithTimeout(60 * time.Second)
}

func waitEnterPressWithTimeout(timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	// 等待 Enter 释放（防抖，避免上次按键残留）
	for time.Now().Before(deadline) {
		if !isEnterDown() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !time.Now().Before(deadline) {
		return fmt.Errorf("等待 Enter 释放超时（%v）", timeout)
	}
	// 等待 Enter 按下
	for time.Now().Before(deadline) {
		if isEnterDown() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !time.Now().Before(deadline) {
		return fmt.Errorf("等待 Enter 按下超时（%v）", timeout)
	}
	// 等待 Enter 释放
	time.Sleep(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if !isEnterDown() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	return nil
}

// user32Dll 和 getAsyncKeyStateFn 提升为包级变量，避免重复创建 syscall
var (
	user32Dll          = syscall.NewLazyDLL("user32.dll")
	getAsyncKeyStateFn = user32Dll.NewProc("GetAsyncKeyState")
)

// isEnterDown 通过 Win32 GetAsyncKeyState(VK_RETURN) 检测 Enter 键是否按下
func isEnterDown() bool {
	const VK_RETURN = 0x0D
	ret, _, _ := getAsyncKeyStateFn.Call(uintptr(VK_RETURN))
	return (ret & 0x8000) != 0
}

// findBrowser 查找 Chrome/Edge 浏览器路径
func findBrowser() string {
	paths := []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`C:\Program Files\Microsoft\Edge\Application\msedge.exe`,
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}
