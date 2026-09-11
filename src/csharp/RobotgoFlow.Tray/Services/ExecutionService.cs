using Microsoft.Extensions.Logging;
using RobotgoFlow.Core.Models;
using RobotgoFlow.Core.Services;
using RobotgoFlow.Tray.ViewModels;

namespace RobotgoFlow.Tray.Services;

/// <summary>
///     工作流执行服务：把引擎事件翻译为进度浮窗 / 托盘通知所需的 UI 更新。
///     只负责订阅与转译，不负责创建或销毁 <see cref="IEngineService"/>——
///     引擎（尤其是 <c>RobotgoNative</c> 单例）由应用级持有者管理生命周期。
/// </summary>
public class ExecutionService : IDisposable
{
    private readonly ILogger _logger;
    private readonly ProgressViewModel _progressVm;
    private readonly ProgressOverlay _overlay;
    private readonly IEngineService _engine;

    private volatile bool _isRunning;
    private volatile bool _disposed;

    public ExecutionService(
        ILogger<ExecutionService> logger,
        ProgressViewModel progressVm,
        ProgressOverlay overlay,
        IEngineService engine)
    {
        _logger = logger;
        _progressVm = progressVm;
        _overlay = overlay;
        _engine = engine;
        _engine.OnEvent += HandleEvent;
    }

    public bool IsRunning => _isRunning;

    public void Start(string workflowPath, string workflowName, int fromStep, string? inputsJson)
    {
        if (_disposed || _isRunning) return;
        // 引擎仍在执行上一次任务时不要启动，否则浮窗会永远停留在“启动中”
        if (_engine.IsRunning)
        {
            _logger.LogWarning("引擎仍在执行上一个工作流，忽略本次启动请求");
            return;
        }
        _isRunning = true;

        _overlay.Dispatcher.InvokeAsync(() =>
        {
            _progressVm.UpdateStep(0, 1, $"启动中: {workflowName}", 0);
            _overlay.Show();
        });

        Task.Run(() =>
        {
            try
            {
                // DLL 模式下 ExecuteWithInputs 立即返回，执行在引擎内部的任务中继续；
                // 子进程模式下同样只是投递执行请求。
                _engine.ExecuteWithInputs(workflowPath, fromStep, false, inputsJson);
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "执行异常");
                _isRunning = false;
                _overlay.Dispatcher.InvokeAsync(async () =>
                {
                    _progressVm.MarkError(ex.Message, null);
                    await Task.Delay(ErrorHideDelayMs);
                    if (!_disposed)
                        _ = _overlay.Dispatcher.InvokeAsync(() => _overlay.Hide());
                });
            }
        });
    }

    private const int SuccessHideDelayMs = 2500;
    private const int ErrorHideDelayMs = 5000;

    public void Dispose()
    {
        if (_disposed) return;
        _disposed = true;
        _engine.OnEvent -= HandleEvent;
        // 注意：此处刻意不调用 _engine.Dispose()。
        // RobotgoNative 是进程级单例，被释放后无法再次初始化，
        // 会导致第二次执行时抛出 ObjectDisposedException。
    }

    public void Stop()
    {
        _engine.Stop();
        _isRunning = false;
    }

    /// <summary>延时后隐藏浮窗（延时期间已销毁则跳过）。</summary>
    private void HideOverlayAfter(int delayMs)
    {
        _overlay.Dispatcher.InvokeAsync(async () =>
        {
            await Task.Delay(delayMs);
            if (!_disposed)
                _ = _overlay.Dispatcher.InvokeAsync(() => _overlay.Hide());
        });
    }

    private void HandleEvent(ServeEvent ev)
    {
        if (_disposed) return;
        _overlay.Dispatcher.InvokeAsync(() =>
        {
            switch (ev.Type)
            {
                case "step_start":
                    _progressVm.UpdateStep(ev.Idx, ev.Total, ev.Name ?? "", ev.EstimatedSec);
                    break;

                case "action_start":
                    _progressVm.UpdateAction(ev.Action ?? "", ev.Detail ?? "");
                    break;

                case "step_done":
                    if (ev.Error != null)
                    {
                        _progressVm.MarkError(ev.Error, ev.ScreenshotPath);
                    }
                    break;

                case "log" when ev.Level == "error":
                    _progressVm.MarkError(ev.Message, null);
                    break;

                case "workflow_done":
                    _isRunning = false;
                    if (ev.Ok == true)
                    {
                        _progressVm.MarkComplete(ev.TotalElapsedSec);
                        HideOverlayAfter(SuccessHideDelayMs);
                        ToastNotifier.ShowSuccess(ev.Name ?? "", ev.TotalSteps, ev.TotalElapsedSec);
                    }
                    else
                    {
                        _progressVm.MarkError(ev.Error ?? "未知错误", ev.ScreenshotPath);
                        ToastNotifier.ShowFailure(ev.Error ?? "未知错误", ev.ScreenshotPath);
                        HideOverlayAfter(ErrorHideDelayMs);
                    }
                    break;

                case "error":
                    _isRunning = false;
                    _progressVm.MarkError(ev.Message ?? "未知错误", null);
                    ToastNotifier.ShowFailure(ev.Message ?? "未知错误", null);
                    HideOverlayAfter(ErrorHideDelayMs);
                    break;

                case "stopped":
                    _isRunning = false;
                    _progressVm.MarkStopped();
                    HideOverlayAfter(SuccessHideDelayMs);
                    break;
            }
        });
    }
}
