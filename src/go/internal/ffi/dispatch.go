// Package ffi 提供 Go 引擎的 C 共享库导出层。
// 所有 robotgo API 调用必须通过 dispatch 串行化到专用 goroutine。
package main

import "robotgo-flow/internal/capture"

// task 表示一个提交到专用线程的工作单元。
type task struct {
	fn   func() string // 实际工作函数，返回 JSON 字符串
	done chan string   // 阻塞等待结果
}

// taskCh 缓冲容量 1，保证同时只有一个任务在执行。
var taskCh = make(chan task, 1)

func init() {
	go func() {
		for t := range taskCh {
			t.done <- t.fn()
		}
	}()

	// 录制器（recorder 包）在独立 goroutine 中调用 capture，capture 内的 robotgo 调用
	// 必须遵循本包的串行化约定，否则会与 dispatch goroutine 并发进入 CGo 层。
	capture.SetSerialRunner(dispatchVoid)
}

// dispatch 将 fn 提交到专用 goroutine 并阻塞等待结果。
// fn 中不得直接或间接调用 dispatch（防止死锁）。
func dispatch(fn func() string) string {
	t := task{fn: fn, done: make(chan string, 1)}
	taskCh <- t
	return <-t.done
}

// dispatchVoid 与 dispatch 相同，但用于没有返回值的场景。
func dispatchVoid(fn func()) {
	dispatch(func() string {
		fn()
		return ""
	})
}
