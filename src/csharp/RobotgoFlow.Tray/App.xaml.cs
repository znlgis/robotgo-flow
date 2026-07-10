using System.Windows;
using Microsoft.Extensions.Logging;
using RobotgoFlow.Core.Services;
using Application = System.Windows.Application;

namespace RobotgoFlow.Tray;

public partial class App : Application
{
    private const string MutexName = "RobotgoFlow.Tray.SingleInstance";
    private static readonly ILoggerFactory s_loggerFactory =
        LoggerFactory.Create(b => b.AddDebug().SetMinimumLevel(LogLevel.Information));
    private static readonly ILogger s_logger = s_loggerFactory.CreateLogger("Tray");

    private System.Threading.Mutex? _mutex;

    protected override void OnStartup(StartupEventArgs e)
    {
        ShutdownMode = ShutdownMode.OnExplicitShutdown;

        _mutex = new System.Threading.Mutex(false, MutexName, out var createdNew);
        if (!createdNew)
        {
            s_logger.LogWarning("检测到已有实例运行");
            ToastNotifier.ShowAlreadyRunning();
            _mutex?.Close();
            Environment.Exit(0);
            return;
        }

        RobotgoNative.DispatchCallback = act => Current.Dispatcher.InvokeAsync(act);

        s_logger.LogInformation("DLL 就绪: {available}", RobotgoNative.IsAvailable);

        var trayIcon = new TrayIcon(
            s_loggerFactory.CreateLogger("TrayIcon"),
            s_loggerFactory);
        trayIcon.Initialize();
    }

    protected override void OnExit(ExitEventArgs e)
    {
        _mutex?.Close();
        base.OnExit(e);
    }
}
