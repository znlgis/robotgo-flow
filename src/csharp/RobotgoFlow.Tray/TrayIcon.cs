using System.IO;
using System.Windows;
using Microsoft.Extensions.Logging;
using Microsoft.Win32;
using RobotgoFlow.Core.Services;
using RobotgoFlow.Tray.Services;
using RobotgoFlow.Tray.ViewModels;
using Application = System.Windows.Application;

namespace RobotgoFlow.Tray;

public class TrayIcon
{
    private readonly ILogger _logger;
    private readonly ILoggerFactory _loggerFactory;
    private NotifyIcon? _notifyIcon;
    private MiniPanelWindow? _miniPanel;
    private ProgressOverlay? _progressOverlay;
    private ExecutionService? _execution;

    public TrayIcon(ILogger logger, ILoggerFactory loggerFactory)
    {
        _logger = logger;
        _loggerFactory = loggerFactory;
    }

    public void Initialize()
    {
        if (!RobotgoNative.IsAvailable)
        {
            _logger.LogError("DLL 加载失败，托盘启动中止");
            ToastNotifier.ShowEngineError("robotgo-flow.dll 加载失败，请检查文件完整性");
            Environment.Exit(1);
            return;
        }

        _progressOverlay = new ProgressOverlay();

        _notifyIcon = new NotifyIcon
        {
            Icon = SystemIcons.Application,
            Text = "robotgo-flow",
            Visible = true,
            ContextMenuStrip = new ContextMenuStrip()
        };

        BuildMenu();
        _logger.LogInformation("托盘图标就绪");
    }

    private void BuildMenu()
    {
        var menu = _notifyIcon!.ContextMenuStrip!;
        menu.Items.Clear();

        var runItem = new ToolStripMenuItem("运行工作流…");
        runItem.Click += (_, _) => ShowMiniPanel();
        menu.Items.Add(runItem);

        var stopItem = new ToolStripMenuItem("停止");
        stopItem.Click += (_, _) =>
        {
            // 仅停止当前执行：引擎为进程级单例，不可在此释放
            _execution?.Stop();
            _progressOverlay?.Hide();
        };
        stopItem.Enabled = false;
        menu.Items.Add(stopItem);

        _notifyIcon.ContextMenuStrip!.Opening += (_, _) =>
        {
            stopItem.Enabled = _execution?.IsRunning == true;
        };

        menu.Items.Add(new ToolStripSeparator());

        var startupItem = new ToolStripMenuItem("开机启动")
        {
            Checked = IsStartupEnabled()
        };
        startupItem.Click += (_, _) =>
        {
            startupItem.Checked = !startupItem.Checked;
            SetStartup(startupItem.Checked);
        };
        menu.Items.Add(startupItem);

        var aboutItem = new ToolStripMenuItem("关于");
        aboutItem.Click += (_, _) =>
            System.Windows.MessageBox.Show("robotgo-flow 托盘执行程序\n.NET 10 WPF\n基于 Go RPA 引擎",
                "关于 robotgo-flow", MessageBoxButton.OK, MessageBoxImage.Information);
        menu.Items.Add(aboutItem);

        menu.Items.Add(new ToolStripSeparator());

        var exitItem = new ToolStripMenuItem("退出");
        exitItem.Click += (_, _) =>
        {
            Shutdown();
            Application.Current.Shutdown();
        };
        menu.Items.Add(exitItem);
    }

    /// <summary>释放托盘资源；可重复调用。</summary>
    private void Shutdown()
    {
        _execution?.Stop();
        _execution?.Dispose();
        _execution = null;
        _progressOverlay?.Close();
        _miniPanel?.Close();
        if (_notifyIcon is not null)
        {
            _notifyIcon.Visible = false;
            _notifyIcon.Dispose();
            _notifyIcon = null;
        }
        // 引擎单例在此统一释放（应用退出）
        RobotgoNative.Instance.Dispose();
    }

    private void ShowMiniPanel()
    {
        if (_miniPanel == null)
        {
            var vm = new MiniPanelViewModel(
                _loggerFactory.CreateLogger("MiniPanel"),
                new FileDialogService());

            // 执行服务复用同一个实例：它只订阅引擎事件，
            // 引擎（RobotgoNative 单例）生命周期由本类持有。
            _execution ??= new ExecutionService(
                _loggerFactory.CreateLogger<ExecutionService>(),
                _progressOverlay!.ViewModel,
                _progressOverlay,
                RobotgoNative.Instance);

            vm.OnStartRequested += (path, name, fromStep, inputsJson) =>
            {
                if (string.IsNullOrEmpty(path)) return;
                _execution.Start(path, name, fromStep, inputsJson);
            };

            _miniPanel = new MiniPanelWindow(vm);
            _miniPanel.Closed += (_, _) => _miniPanel = null;
        }

        _miniPanel.ShowNearTray();
    }

    private static bool IsStartupEnabled()
    {
        using var key = Registry.CurrentUser.OpenSubKey(
            @"Software\Microsoft\Windows\CurrentVersion\Run", false);
        return key?.GetValue("robotgo-flow-tray") != null;
    }

    private static void SetStartup(bool enable)
    {
        var exePath = Environment.ProcessPath
            ?? System.Diagnostics.Process.GetCurrentProcess().MainModule?.FileName;
        if (string.IsNullOrWhiteSpace(exePath)) return;
        // 拒绝临时目录，避免将不可信路径写入注册表
        var tempPath = Path.GetTempPath();
        if (exePath.StartsWith(tempPath, StringComparison.OrdinalIgnoreCase)) return;

        using var key = Registry.CurrentUser.OpenSubKey(
            @"Software\Microsoft\Windows\CurrentVersion\Run", true);
        if (enable)
            key?.SetValue("robotgo-flow-tray", exePath);
        else
            key?.DeleteValue("robotgo-flow-tray", false);
    }
}
