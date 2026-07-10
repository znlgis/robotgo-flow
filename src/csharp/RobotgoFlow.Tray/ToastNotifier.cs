using CommunityToolkit.WinUI.Notifications;

namespace RobotgoFlow.Tray;

public static class ToastNotifier
{
    /// <summary>成功完成通知</summary>
    public static void ShowSuccess(string workflowName, int totalSteps, double elapsedSec)
    {
        ShowToastSafely(() => new ToastContentBuilder()
            .AddArgument("action", "none")
            .AddText($"✅ 执行完成 — {workflowName}")
            .AddText($"{totalSteps} 个步骤全部成功 · 耗时 {elapsedSec:F1} 秒")
            .Show());
    }

    /// <summary>失败通知（可交互）</summary>
    public static void ShowFailure(string errorMessage, string? screenshotPath)
    {
        ShowToastSafely(() => new ToastContentBuilder()
            .AddArgument("action", "viewDetail")
            .AddArgument("screenshotPath", screenshotPath ?? "")
            .AddText("❌ 执行失败")
            .AddText(errorMessage)
            .AddButton(new ToastButton()
                .SetContent("查看详情")
                .AddArgument("action", "viewDetail"))
            .AddButton(new ToastButton()
                .SetContent("重试")
                .AddArgument("action", "retry"))
            .Show());
    }

    /// <summary>引擎加载失败通知</summary>
    public static void ShowEngineError(string message)
    {
        ShowToastSafely(() => new ToastContentBuilder()
            .AddArgument("action", "engineError")
            .AddText("⚠️ 引擎加载失败")
            .AddText(message)
            .Show());
    }

    /// <summary>已在运行通知</summary>
    public static void ShowAlreadyRunning()
    {
        ShowToastSafely(() => new ToastContentBuilder()
            .AddArgument("action", "alreadyRunning")
            .AddText("robotgo-flow")
            .AddText("已在运行中，请查看系统托盘图标")
            .Show());
    }

    /// <summary>
    ///     统一包装 Toast 发送逻辑，捕获异常避免影响主流程。
    ///     提取公共方法以减少各通知方法中的重复 try-catch 代码。
    /// </summary>
    private static void ShowToastSafely(Action showToast)
    {
        try
        {
            showToast();
        }
        catch (System.Exception ex)
        {
            System.Diagnostics.Trace.WriteLine($"[ToastNotifier] 通知发送失败: {ex.Message}");
        }
    }
}
