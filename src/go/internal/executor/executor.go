package executor

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"robotgo-flow/internal/action"
	"robotgo-flow/internal/config"
	"robotgo-flow/internal/logger"
	"robotgo-flow/internal/notify"
)

// ProgressCallback 接收实时执行进度通知。
// 所有方法均在执行器的 goroutine 中调用。
// 实现者需通过 UI 框架的线程调度方法（如 WPF 的 Dispatcher.Invoke）
// 执行 UI 线程操作。
type ProgressCallback interface {
	OnWorkflowStart(name string, totalSteps int)
	OnStepStart(stepIdx, totalSteps int, stepName string, estimatedSec float64)
	OnActionStart(stepIdx, actionIdx int, actionType string, desc string)
	OnActionDone(stepIdx, actionIdx int)
	OnStepDone(stepIdx int, err error, screenshotPath string)
	OnWorkflowDone(name string, totalSteps int, err error, totalElapsedSec float64)
	OnLog(level LogLevel, message string, stepIdx int)
}

// LogLevel 表示日志消息的严重程度。
type LogLevel int

const (
	LogInfo LogLevel = iota
	LogWarn
	LogError
)

// Executor 工作流执行器
type Executor struct {
	eng         action.Engine
	cfg         *config.Workflow
	workflowDir string
	callback    ProgressCallback // 可选：nil 表示无回调（向后兼容）
	ctx         context.Context  // 可选：nil 表示不支持取消

	startTime      time.Time
	currentStepIdx int
	debugScreens   bool
}

// New 创建执行器
func New(eng action.Engine, cfg *config.Workflow, workflowPath string) *Executor {
	return &Executor{
		eng:            eng,
		cfg:            cfg,
		workflowDir:    filepath.Dir(workflowPath),
		ctx:            context.Background(),
		currentStepIdx: -1,
	}
}

// WithContext 设置执行上下文（非并发安全，应在创建后、Run 之前调用一次）。
// 传入 nil 时自动替换为 context.Background()，保证 e.ctx 永远不会是 nil。
func (e *Executor) WithContext(ctx context.Context) *Executor {
	if ctx == nil {
		e.ctx = context.Background()
	} else {
		e.ctx = ctx
	}
	return e
}

// ErrCancelled 表示执行被用户请求停止。
var ErrCancelled = fmt.Errorf("执行已被用户取消")

// SetCallback 注册 ProgressCallback 以接收执行进度。
// 传入 nil 可禁用回调。可在 Run/RunFromStep 之前安全调用。
func (e *Executor) SetCallback(cb ProgressCallback) {
	e.callback = cb
}

// EnableDebugScreenshots 控制是否在每个成功步骤结束后自动保存调试截图。
func (e *Executor) EnableDebugScreenshots(enabled bool) {
	e.debugScreens = enabled
}

func (e *Executor) log(level LogLevel, format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	switch level {
	case LogWarn:
		logger.Warn("%s", msg)
	case LogError:
		logger.Error("%s", msg)
	default:
		logger.Info("%s", msg)
	}
	if e.callback != nil {
		e.callback.OnLog(level, msg, e.currentStepIdx)
	}
}

// Run 执行完整工作流
func (e *Executor) Run() error {
	return e.runSteps(0)
}

// RunFromStep 从指定步骤开始执行（1-indexed）
func (e *Executor) RunFromStep(startStep int) error {
	if startStep < 1 || startStep > len(e.cfg.Steps) {
		return fmt.Errorf("步骤编号 %d 超出范围 [1, %d]", startStep, len(e.cfg.Steps))
	}
	logger.Info("[Executor] 从步骤 %d 开始执行工作流: %s", startStep, e.cfg.Name)
	return e.runSteps(startStep - 1)
}

// ResolveInputs 将工作流中所有 $input.<name> 占位符替换为对应采集值。
// 此方法直接操作 cfg.Steps，不依赖引擎实例。
func (e *Executor) ResolveInputs(values map[string]string) {
	ResolveInputs(e.cfg, values)
}

// ResolveInputs 替换 cfg 中所有 $input.<name> 占位符为对应值。
// 供 serve 等场景在未创建引擎实例前安全调用。
// 覆盖 Type、OpenURL、Prompt、Confirm、Notify 等可能包含占位符的字段。
func ResolveInputs(cfg *config.Workflow, values map[string]string) {
	if values == nil || len(values) == 0 {
		return
	}

	for i := range cfg.Steps {
		for j := range cfg.Steps[i].Actions {
			action := &cfg.Steps[i].Actions[j]
			if action.Type != nil {
				action.Type.Text = expandVars(action.Type.Text, values)
				action.Type.Into = expandVars(action.Type.Into, values)
			}
			if action.OpenURL != "" {
				action.OpenURL = expandVars(action.OpenURL, values)
			}
			if action.Prompt != nil {
				action.Prompt.Title = expandVars(action.Prompt.Title, values)
				action.Prompt.Message = expandVars(action.Prompt.Message, values)
				action.Prompt.Into = expandVars(action.Prompt.Into, values)
			}
			if action.Confirm != nil {
				action.Confirm.Title = expandVars(action.Confirm.Title, values)
				action.Confirm.Message = expandVars(action.Confirm.Message, values)
			}
			if action.Notify != nil {
				action.Notify.Title = expandVars(action.Notify.Title, values)
				action.Notify.Message = expandVars(action.Notify.Message, values)
			}
		}
	}
}

// expandVars 将 s 中的 $input.<name> 占位符替换为 vars 中的值。
// 使用 strings.NewReplacer 进行单次扫描替换，按变量名长度降序排列防止短名称误匹配长名称前缀。
func expandVars(s string, vars map[string]string) string {
	// 按名称长度降序排列，防止短名称误匹配长名称的前缀
	names := make([]string, 0, len(vars))
	for name := range vars {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return len(names[i]) > len(names[j])
	})

	pairs := make([]string, 0, len(vars)*2)
	for _, name := range names {
		pairs = append(pairs, "$input."+name, vars[name])
	}
	return strings.NewReplacer(pairs...).Replace(s)
}

// describeAction 返回动作类型与中文描述，供 ProgressCallback 展示。
// 动作类型未知时返回 ("unknown", "")。
func describeAction(cfg config.Action) (actionType, detail string) {
	t := cfg.ActionType()
	if t == "" {
		return "unknown", ""
	}
	return t, cfg.Label()
}

// runSteps 从指定索引开始执行所有步骤。
func (e *Executor) runSteps(startIndex int) error {
	e.startTime = time.Now()
	e.currentStepIdx = startIndex
	e.log(LogInfo, "[Executor] 开始执行工作流: %s", e.cfg.Name)

	if e.callback != nil {
		e.callback.OnWorkflowStart(e.cfg.Name, len(e.cfg.Steps))
	}

	if err := e.eng.FindBrowserWindow(); err != nil {
		e.log(LogWarn, "[Executor] 未找到浏览器窗口: %v (将在全屏模式下运行)", err)
	}

	for i := startIndex; i < len(e.cfg.Steps); i++ {
		e.currentStepIdx = i
		// 步骤间检查取消信号
		select {
		case <-e.ctx.Done():
			return ErrCancelled
		default:
		}

		step := e.cfg.Steps[i]
		e.log(LogInfo, "━━━ 步骤 %d/%d: %s ━━━", i+1, len(e.cfg.Steps), step.Name)

		if e.callback != nil {
			e.callback.OnStepStart(i, len(e.cfg.Steps), step.Name, 0)
		}

		for j, actionCfg := range step.Actions {
			// 每个动作前检查取消信号
			select {
			case <-e.ctx.Done():
				return ErrCancelled
			default:
			}

			act, err := action.FromConfig(actionCfg, e.cfg.Settings.ElementTimeout)
			if err != nil {
				if e.callback != nil {
					e.callback.OnStepDone(i, err, "")
					e.callback.OnWorkflowDone(e.cfg.Name, len(e.cfg.Steps), err, time.Since(e.startTime).Seconds())
				}
				return fmt.Errorf("步骤 %q 动作 %d 解析失败: %w", step.Name, j+1, err)
			}

			if e.callback != nil {
				actionType, detail := describeAction(actionCfg)
				e.callback.OnActionStart(i, j, actionType, detail)
			}

			// 根据 OnError 设置确定最大尝试次数（含首次执行）
			retries := 1
			if e.cfg.Settings.OnError == "retry" {
				retries = e.cfg.Settings.MaxRetries
				if retries < 1 {
					retries = 1
				}
			}

			var execErr error
			for attempt := 0; attempt < retries; attempt++ {
				// 重试过程中检查取消信号
				select {
				case <-e.ctx.Done():
					return ErrCancelled
				default:
				}

				execErr = act.Execute(e.eng)
				if execErr == nil {
					if e.callback != nil {
						e.callback.OnActionDone(i, j)
					}
					break // 成功
				}

				// 重试逻辑
				if e.cfg.Settings.OnError == "retry" && attempt < retries-1 {
					e.log(LogWarn, "[Retry] 步骤 %q 动作 %d 第 %d/%d 次尝试失败，1 秒后重试: %v",
						step.Name, j+1, attempt+1, retries, execErr)
					e.eng.Wait(1000) // 重试前短暂暂停
					continue
				}

				// 跳过逻辑 —— 记录日志并继续执行下一个动作
				if e.cfg.Settings.OnError == "skip" {
					e.log(LogWarn, "[Skip] 步骤 %q 动作 %d 已跳过: %v", step.Name, j+1, execErr)
					execErr = nil // 清除错误以继续
					// 保持回调协议对称：OnActionStart 之后必须有对应的 OnActionDone
					if e.callback != nil {
						e.callback.OnActionDone(i, j)
					}
					break
				}

				// 中断（默认）或重试耗尽 —— 通知并返回错误
				filename := fmt.Sprintf("error_step%d_action%d.png", i+1, j+1)
				screenshotPath := e.eng.CaptureScreen(filename)

				// 写入错误日志文件
				notify.LogError(e.workflowDir, step.Name, j+1, execErr, screenshotPath)

				// 弹出错误对话框
				notify.PrintError("robotgo-flow 执行失败",
					fmt.Sprintf("步骤 [%s] 第 %d 个动作失败\n错误: %v\n截图已保存: %s",
						step.Name, j+1, execErr, screenshotPath))

				if e.callback != nil {
					e.callback.OnStepDone(i, execErr, screenshotPath)
					e.callback.OnWorkflowDone(e.cfg.Name, len(e.cfg.Steps), execErr, time.Since(e.startTime).Seconds())
				}
				return fmt.Errorf("步骤 %q 动作 %d 执行失败: %w\n已保存错误截图: %s",
					step.Name, j+1, execErr, screenshotPath)
			}
		}

		screenshotPath := ""
		if e.debugScreens {
			screenshotPath = e.eng.CaptureScreen(fmt.Sprintf("debug_step_%03d.png", i+1))
		}
		if e.callback != nil {
			e.callback.OnStepDone(i, nil, screenshotPath)
		}
	}

	if e.callback != nil {
		e.callback.OnWorkflowDone(e.cfg.Name, len(e.cfg.Steps), nil, time.Since(e.startTime).Seconds())
	}

	e.log(LogInfo, "✓ 工作流执行完成")
	return nil
}
