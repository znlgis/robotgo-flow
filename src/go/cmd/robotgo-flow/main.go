package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"robotgo-flow/internal/capture"
	"robotgo-flow/internal/config"
	"robotgo-flow/internal/engine"
	"robotgo-flow/internal/executor"
	"robotgo-flow/internal/notify"
	"robotgo-flow/internal/recorder"
	"robotgo-flow/internal/serve"
)

// Version 通过构建时 -ldflags 注入。
// 示例: go build -ldflags "-X main.Version=1.2.3" ./cmd/robotgo-flow
var Version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "run":
		cmdRun(os.Args[2:])
	case "record":
		cmdRecord(os.Args[2:])
	case "capture":
		cmdCapture(os.Args[2:])
	case "serve":
		cmdServe(os.Args[2:])
	case "version", "-v", "--version":
		fmt.Printf("robotgo-flow version %s\n", Version)
	case "-h", "--help", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "未知子命令: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("robotgo-flow — Windows 桌面 RPA 自动化工具")
	fmt.Println()
	fmt.Println("用法:")
	fmt.Println("  robotgo-flow                      显示帮助信息")
	fmt.Println("  robotgo-flow version              显示版本号")
	fmt.Println("  robotgo-flow run <工作流文件>      执行工作流")
	fmt.Println("  robotgo-flow record               交互式录制工作流")
	fmt.Println("  robotgo-flow capture [元素名]     交互式截图模板")
	fmt.Println("  robotgo-flow serve <工作流文件>   启动 JSON-Line 协议服务 (供 GUI 调用)")
	fmt.Println()
	fmt.Println("选项:")
	fmt.Println("  -h, --help                        显示帮助")
}

// ---------- run 子命令 ----------

func cmdRun(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	fromStep := fs.Int("from", 1, "从指定步骤开始执行 (1-indexed)")
	outDir := fs.String("out", "", "截图输出目录 (默认: 工作流文件所在目录)")
	debug := fs.Bool("debug", false, "调试模式：每个成功步骤结束后自动保存截图")

	fs.Usage = func() {
		fmt.Println("用法: robotgo-flow run [选项] <工作流文件>")
		fmt.Println()
		fmt.Println("选项:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "错误: 请指定工作流文件")
		fs.Usage()
		os.Exit(1)
	}

	workflowPath := fs.Arg(0)
	absPath, err := filepath.Abs(workflowPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取工作流文件的绝对路径失败: %v\n", err)
		os.Exit(1)
	}

	// 加载工作流配置
	cfg, err := config.Load(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载工作流失败: %v\n", err)
		os.Exit(1)
	}

	workDir := filepath.Dir(absPath)
	screenshotDir := *outDir
	if screenshotDir == "" {
		screenshotDir = filepath.Join(workDir, "screenshots")
	}

	// 创建引擎
	eng := engine.NewEngine(workDir, screenshotDir)
	defer eng.Close() // 确保引擎资源（模板缓存等）被释放

	// 启用人性化模式
	if cfg.Settings.Human.Enabled {
		eng.EnableHuman(true, cfg.Settings.Human.Speed, cfg.Settings.Human.MistakeRate, true, true, false)
	}

	exe := executor.New(eng, cfg, absPath)
	exe.EnableDebugScreenshots(*debug)

	// 采集运行时变量 (CLI 下使用 stdin 输入)
	if len(cfg.Inputs) > 0 {
		fmt.Println("═══ 采集运行时变量 ═══")
		// 先一次性采集全部变量，再统一替换占位符：
		// 逐项替换会让后输入的变量值被先前值中的 $input. 文本二次替换。
		values := make(map[string]string, len(cfg.Inputs))
		for _, input := range cfg.Inputs {
			val, err := notify.InputBoxStd(input.Label, input.Placeholder, input.Mask)
			if err != nil {
				fmt.Fprintf(os.Stderr, "读取输入 %q 失败: %v\n", input.Label, err)
				os.Exit(1)
			}
			if val == "" && input.Required {
				fmt.Fprintf(os.Stderr, "\n输入 %q 是必填的，不能为空\n", input.Label)
				os.Exit(1)
			}
			values[input.Name] = val
		}
		exe.ResolveInputs(values)
		fmt.Println()
	}

	fmt.Printf("开始执行工作流: %s\n", cfg.Name)

	err = exe.RunFromStep(*fromStep)
	if err != nil {
		fmt.Fprintf(os.Stderr, "执行失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("工作流执行完成")
}

// ---------- record 子命令 ----------

func cmdRecord(args []string) {
	fs := flag.NewFlagSet("record", flag.ExitOnError)
	out := fs.String("out", "workflow.yaml", "输出工作流文件路径")
	tplDir := fs.String("tpl-dir", "templates", "模板截图存放目录")
	pipe := fs.Bool("pipe", false, "JSON-Line 管道模式（供 GUI 调用）")

	fs.Usage = func() {
		fmt.Println("用法: robotgo-flow record [选项]")
		fmt.Println()
		fmt.Println("选项:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	absOut, err := filepath.Abs(*out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取输出文件的绝对路径失败: %v\n", err)
		os.Exit(1)
	}
	absTplDir, err := filepath.Abs(*tplDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取模板目录的绝对路径失败: %v\n", err)
		os.Exit(1)
	}

	if *pipe {
		// 管道模式：通过 JSON-Line 与 GUI 通信
		rec := recorder.NewPipe(absOut, absTplDir, os.Stdin, os.Stdout)
		if err := rec.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "录制失败: %v\n", err)
			os.Exit(1)
		}
	} else {
		// 交互模式：终端 Q&A（现有逻辑不变）
		rec := recorder.New(absOut, absTplDir, os.Stdin)
		if err := rec.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "录制失败: %v\n", err)
			os.Exit(1)
		}
	}
}

// ---------- capture 子命令 ----------

func cmdCapture(args []string) {
	fs := flag.NewFlagSet("capture", flag.ExitOnError)
	outDir := fs.String("out-dir", "templates", "截图输出目录")

	fs.Usage = func() {
		fmt.Println("用法: robotgo-flow capture [选项] [元素名]")
		fmt.Println()
		fmt.Println("选项:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	name := "element"
	if fs.NArg() >= 1 {
		name = fs.Arg(0)
	}

	fmt.Printf("请框选元素: %s\n", name)
	if err := capture.CaptureInteractive(name, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "截图失败: %v\n", err)
		os.Exit(1)
	}
}

// ---------- serve 子命令 ----------

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	fromStep := fs.Int("from", 1, "从指定步骤开始执行 (1-indexed)")
	debug := fs.Bool("debug", false, "调试模式：每个成功步骤结束后自动保存截图")

	fs.Usage = func() {
		fmt.Println("用法: robotgo-flow serve [选项] <工作流文件>")
		fmt.Println()
		fmt.Println("选项:")
		fs.PrintDefaults()
	}
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "用法: robotgo-flow serve <工作流文件>")
		os.Exit(1)
	}
	workflowPath := fs.Arg(0)
	if err := serve.Run(workflowPath, os.Stdin, os.Stdout, *fromStep, *debug); err != nil {
		// 用户主动停止属于正常结束（GUI 已收到 stopped 事件），不应报告为失败
		if errors.Is(err, executor.ErrCancelled) {
			fmt.Fprintln(os.Stderr, "serve: 执行已被用户停止")
			return
		}
		fmt.Fprintf(os.Stderr, "serve: %v\n", err)
		os.Exit(1)
	}
}
