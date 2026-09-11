package notify

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// PrintError 向 stderr 输出格式化的错误消息。
func PrintError(title, text string) {
	fmt.Fprintf(os.Stderr, "\n═══ %s ═══\n%s\n\n", title, text)
}

// Interactor 抽象交互式输入（输入框 / 确认框）。
// 默认实现读取 stdin；serve/DLL 等 stdin 已被协议占用或没有控制台的宿主
// 可通过 SetInteractor / DisableInteraction 替换，避免阻塞或破坏协议流。
type Interactor interface {
	// Input 请求用户输入一行文本。placeholder 为提示文本，mask 表示密码输入。
	Input(label, placeholder string, mask bool) (string, error)
	// Confirm 请求用户确认，返回 true 表示用户选择了“是”。
	Confirm(title, message string) (bool, error)
}

var (
	interactorMu sync.RWMutex
	interactor   Interactor = stdinInteractor{}
)

// SetInteractor 替换全局交互实现。传入 nil 恢复默认的 stdin 交互实现。
func SetInteractor(i Interactor) {
	interactorMu.Lock()
	defer interactorMu.Unlock()
	if i == nil {
		interactor = stdinInteractor{}
		return
	}
	interactor = i
}

// DisableInteraction 安装一个拒绝所有交互动作的实现。
// reason 用于向用户解释原因（例如“serve 模式下 stdin 被 JSON 协议占用”）。
// 这样交互动作会立即返回明确错误，而不是读取协议流或永久阻塞。
func DisableInteraction(reason string) {
	SetInteractor(unsupportedInteractor{reason: reason})
}

// InteractorEnabled 报告当前是否使用默认的 stdin 交互实现。
func InteractorEnabled() bool {
	interactorMu.RLock()
	defer interactorMu.RUnlock()
	_, ok := interactor.(stdinInteractor)
	return ok
}

func currentInteractor() Interactor {
	interactorMu.RLock()
	defer interactorMu.RUnlock()
	return interactor
}

// InputBoxStd 从当前 Interactor 读取一行输入。
func InputBoxStd(label, placeholder string, mask bool) (string, error) {
	return currentInteractor().Input(label, placeholder, mask)
}

// ConfirmBoxStd 从当前 Interactor 读取确认（y/n）。
func ConfirmBoxStd(title, message string) (bool, error) {
	return currentInteractor().Confirm(title, message)
}

// ---------------------------------------------------------------------------
// 默认实现：stdin
// ---------------------------------------------------------------------------

// stdinInteractor 从 stdin 逐行读取输入。
//
// 复用同一个 bufio.Reader 至关重要：bufio.Reader 会一次性从 stdin 预读整块数据，
// 若每次调用都新建 Reader，则前一次预读但未消费的字节会被丢弃，
// 导致连续多次调用时除第一次外全部读不到内容（甚至永久阻塞）。
type stdinInteractor struct{}

var (
	stdinMu     sync.Mutex
	stdinReader = bufio.NewReader(os.Stdin)
)

func (stdinInteractor) Input(label, placeholder string, mask bool) (string, error) {
	prompt := "\n" + label
	if placeholder != "" {
		prompt += fmt.Sprintf(" (%s)", placeholder)
	}
	if mask {
		// 注意: CLI 模式下无法真正隐藏输入（需 golang.org/x/term 支持），
		// 建议密码类输入使用 GUI 模式
		prompt += " [注意: CLI 模式下输入不会隐藏，请使用 GUI 模式输入密码]"
	}
	prompt += ": "
	fmt.Fprint(os.Stderr, prompt)

	line, err := readStdinLine()
	if err != nil {
		return "", err
	}
	return line, nil
}

func (stdinInteractor) Confirm(title, message string) (bool, error) {
	fmt.Fprintf(os.Stderr, "\n[确认] %s: %s (y/n): ", title, message)
	line, err := readStdinLine()
	if err != nil {
		return false, err
	}
	// 仅在比较时忽略大小写，避免影响输入框中的原始文本
	switch strings.ToLower(line) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// readStdinLine 读取 stdin 的一行并去除首尾空白（保留原始大小写）。
func readStdinLine() (string, error) {
	stdinMu.Lock()
	defer stdinMu.Unlock()

	line, err := stdinReader.ReadString('\n')
	if err != nil {
		// 已经读到部分数据（最后一行没有换行符）时仍然返回该行
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return trimmed, nil
		}
		if err == io.EOF {
			return "", fmt.Errorf("输入已结束 (stdin 已关闭)")
		}
		return "", fmt.Errorf("读取输入失败: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// ---------------------------------------------------------------------------
// 非交互宿主实现
// ---------------------------------------------------------------------------

// unsupportedInteractor 在 stdin 不可用的宿主中拒绝所有交互动作。
type unsupportedInteractor struct{ reason string }

func (u unsupportedInteractor) Input(string, string, bool) (string, error) {
	return "", fmt.Errorf("当前运行模式不支持交互式输入 (prompt): %s", u.reason)
}

func (u unsupportedInteractor) Confirm(string, string) (bool, error) {
	return false, fmt.Errorf("当前运行模式不支持交互式确认 (confirm): %s", u.reason)
}

// ---------------------------------------------------------------------------
// 错误日志
// ---------------------------------------------------------------------------

// ErrorLogEntry is the structured error log record.
type ErrorLogEntry struct {
	Time       string `json:"time"`
	Step       string `json:"step"`
	Action     int    `json:"action"`
	Error      string `json:"error"`
	Screenshot string `json:"screenshot"`
}

// LogError appends a structured error entry to the workflow error log file.
func LogError(logDir, stepName string, actionIdx int, err error, screenshotPath string) {
	entry := ErrorLogEntry{
		Time:       time.Now().Format(time.RFC3339),
		Step:       stepName,
		Action:     actionIdx,
		Error:      err.Error(),
		Screenshot: screenshotPath,
	}
	logPath := filepath.Join(logDir, "workflow-error.log")
	f, openErr := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if openErr != nil {
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	_ = enc.Encode(entry)
	fmt.Fprintln(f)
}
