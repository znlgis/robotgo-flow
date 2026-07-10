using Microsoft.Extensions.Logging;
using RobotgoFlow.Core.Models;
using RobotgoFlow.Core.Services;
using RobotgoFlow.Tray.ViewModels;

namespace RobotgoFlow.Tray.Services;

public class ExecutionService : IDisposable
{
    private readonly ILogger _logger;
    private readonly ProgressViewModel _progressVm;
    private readonly ProgressOverlay _overlay;
    private readonly Action<string?, string?, int, string?> _onFinished;
    private readonly IEngineService _engine;

    private volatile bool _isRunning;
    private volatile bool _disposed;

    public ExecutionService(
        ILogger<ExecutionService> logger,
        ProgressViewModel progressVm,
        ProgressOverlay overlay,
        Action<string?, string?, int, string?> onFinished,
        IEngineService engine)
    {
        _logger = logger;
        _progressVm = progressVm;
        _overlay = overlay;
        _onFinished = onFinished;
        _engine = engine;
        _engine.OnEvent += HandleEvent;
    }

    public bool IsRunning => _isRunning;

    public void Start(string workflowPath, string workflowName, int fromStep, string? inputsJson)
    {
        if (_isRunning) return;
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
                _engine.ExecuteWithInputs(workflowPath, fromStep, false, inputsJson);
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "执行异常");
                _isRunning = false;
                _overlay.Dispatcher.InvokeAsync(async () =>
                {
                    _progressVm.MarkError(ex.Message, null);
                    await Task.Delay(5000);
                    if (!_disposed)
                        _ = _overlay.Dispatcher.InvokeAsync(() => _overlay.Hide());
                });
            }
        });
    }

    public void Dispose()
    {
        _disposed = true;
        _engine.OnEvent -= HandleEvent;
        _engine.Dispose();
    }

    public void Stop()
    {
        _engine.Stop();
        _isRunning = false;
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
                        _overlay.Dispatcher.InvokeAsync(async () =>
                        {
                            await Task.Delay(2500);
                            if (!_disposed)
                                _ = _overlay.Dispatcher.InvokeAsync(() => _overlay.Hide());
                        });
                        ToastNotifier.ShowSuccess(ev.Name ?? "", ev.TotalSteps, ev.TotalElapsedSec);
                    }
                    else
                    {
                        _progressVm.MarkError(ev.Error ?? "未知错误", ev.ScreenshotPath);
                        ToastNotifier.ShowFailure(ev.Error ?? "未知错误", ev.ScreenshotPath);
                        _overlay.Dispatcher.InvokeAsync(async () =>
                        {
                            await Task.Delay(5000);
                            if (!_disposed)
                                _ = _overlay.Dispatcher.InvokeAsync(() => _overlay.Hide());
                        });
                    }
                    _onFinished?.Invoke(ev.Error, ev.ScreenshotPath,
                        (int)ev.TotalElapsedSec, null);
                    break;

                case "error":
                    _isRunning = false;
                    _progressVm.MarkError(ev.Message ?? "未知错误", null);
                    ToastNotifier.ShowFailure(ev.Message ?? "未知错误", null);
                    _overlay.Dispatcher.InvokeAsync(async () =>
                    {
                        await Task.Delay(5000);
                        if (!_disposed)
                            _ = _overlay.Dispatcher.InvokeAsync(() => _overlay.Hide());
                    });
                    _onFinished?.Invoke(ev.Message, null, 0, null);
                    break;

                case "stopped":
                    _isRunning = false;
                    break;
            }
        });
    }
}
