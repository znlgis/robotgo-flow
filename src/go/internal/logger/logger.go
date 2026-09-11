// Package logger 提供简单的分级日志输出，封装标准库 log 包。
// 未来可替换为结构化日志实现（如 slog/zap）而无需改动调用方。
package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync/atomic"
)

// Level 日志级别
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger 分级日志记录器
type Logger struct {
	level  atomic.Int32 // 通过原子操作保护，支持并发读写
	debug  *log.Logger
	info   *log.Logger
	warn   *log.Logger
	errLog *log.Logger
}

// DefaultLogger 全局默认日志记录器
var DefaultLogger = New(LevelInfo)

// New 创建新的 Logger（Info/Warn/Debug 输出到 stdout，Error 输出到 stderr）
func New(level Level) *Logger {
	l := &Logger{
		debug:  log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile),
		info:   log.New(os.Stdout, "[INFO] ", log.LstdFlags),
		warn:   log.New(os.Stdout, "[WARN] ", log.LstdFlags),
		errLog: log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
	}
	l.level.Store(int32(level))
	return l
}

// SetLevel 修改日志级别（并发安全）
func (l *Logger) SetLevel(level Level) { l.level.Store(int32(level)) }

// Level 返回当前日志级别（并发安全）
func (l *Logger) Level() Level { return Level(l.level.Load()) }

// SetOutput 将 Debug/Info/Warn 重定向到 w（Error 始终写 stderr）。
// log.Logger.SetOutput 自身并发安全，可随时调用。
// 典型用途：serve 模式下 stdout 承载 JSON-Line 协议，日志必须改道 stderr，
// 否则日志行会混入协议流。
func (l *Logger) SetOutput(w io.Writer) {
	l.debug.SetOutput(w)
	l.info.SetOutput(w)
	l.warn.SetOutput(w)
}

// Debug 调试信息，仅在 level <= LevelDebug 时输出
func (l *Logger) Debug(format string, args ...interface{}) {
	if Level(l.level.Load()) > LevelDebug {
		return
	}
	l.debug.Output(2, fmt.Sprintf(format, args...))
}

// Info 信息日志
func (l *Logger) Info(format string, args ...interface{}) {
	if Level(l.level.Load()) > LevelInfo {
		return
	}
	l.info.Output(2, fmt.Sprintf(format, args...))
}

// Warn 警告日志
func (l *Logger) Warn(format string, args ...interface{}) {
	if Level(l.level.Load()) > LevelWarn {
		return
	}
	l.warn.Output(2, fmt.Sprintf(format, args...))
}

// Error 错误日志
func (l *Logger) Error(format string, args ...interface{}) {
	l.errLog.Output(2, fmt.Sprintf(format, args...))
}

// ---- 包级便捷函数，使用 DefaultLogger ----

func Debug(format string, args ...interface{}) { DefaultLogger.Debug(format, args...) }
func Info(format string, args ...interface{})  { DefaultLogger.Info(format, args...) }
func Warn(format string, args ...interface{})  { DefaultLogger.Warn(format, args...) }
func Error(format string, args ...interface{}) { DefaultLogger.Error(format, args...) }

// SetOutput 将默认日志器的 Debug/Info/Warn 重定向到 w，返回恢复函数。
// 用法: defer logger.SetOutput(os.Stderr)()
func SetOutput(w io.Writer) func() {
	DefaultLogger.SetOutput(w)
	return func() { DefaultLogger.SetOutput(os.Stdout) }
}
