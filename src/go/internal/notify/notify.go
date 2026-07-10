package notify

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PrintError 向 stderr 输出格式化的错误消息。
func PrintError(title, text string) {
	fmt.Fprintf(os.Stderr, "\n═══ %s ═══\n%s\n\n", title, text)
}

// ConfirmBoxStd 从 stdin 读取单字符确认（y/n）。
// 返回 true 表示输入了 "y" 或 "yes"（不区分大小写），否则返回 false。
func ConfirmBoxStd(title, message string) bool {
	fmt.Fprintf(os.Stderr, "\n[确认] %s: %s (y/n): ", title, message)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes"
}

// InputBoxStd 从 stdin 读取一行输入。
func InputBoxStd(label, placeholder string, mask bool) string {
	prompt := fmt.Sprintf("\n%s", label)
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

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return ""
	}
	return strings.TrimSpace(line)
}

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
