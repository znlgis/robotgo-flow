using CommunityToolkit.Mvvm.ComponentModel;

namespace RobotgoFlow.Tray.ViewModels;

public partial class ProgressViewModel : ObservableObject
{
    [ObservableProperty] private string _stepName = "就绪";
    [ObservableProperty] private int _currentStep;
    [ObservableProperty] private int _totalSteps;
    [ObservableProperty] private string _currentAction = "";
    [ObservableProperty] private double _estimatedRemainingSec;
    [ObservableProperty] private string _estimatedText = "";
    [ObservableProperty] private bool _hasError;
    [ObservableProperty] private bool _isComplete;
    [ObservableProperty] private double _progress; // 0.0 ~ 1.0
    [ObservableProperty] private string _progressText = "0%";
    [ObservableProperty] private string _status = "running"; // "running" / "success" / "error"

    public void UpdateStep(int idx, int total, string name, double estimatedSec)
    {
        CurrentStep = idx + 1;
        TotalSteps = total;
        StepName = name;
        Progress = (double)CurrentStep / total;
        ProgressText = $"{(int)(Progress * 100)}%";
        if (estimatedSec > 0)
        {
            EstimatedRemainingSec = estimatedSec * (total - CurrentStep + 1);
            EstimatedText = EstimatedRemainingSec < 60
                ? $"预计剩余 {EstimatedRemainingSec:F0} 秒"
                : $"预计剩余 {EstimatedRemainingSec / 60:F1} 分钟";
        }
        HasError = false;
        Status = "running";
    }

    public void UpdateAction(string action, string detail)
    {
        CurrentAction = $"{action}: {detail}";
    }

    public void MarkError(string? errorMessage, string? screenshotPath)
    {
        HasError = true;
        Status = "error";
        CurrentAction = errorMessage ?? "未知错误";
    }

    public void MarkComplete(double totalElapsedSec)
    {
        IsComplete = true;
        Status = "success";
        Progress = 1.0;
        ProgressText = "100%";
        EstimatedText = $"完成 · 耗时 {totalElapsedSec:F1} 秒";
    }
}
