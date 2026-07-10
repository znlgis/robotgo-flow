using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.IO;
using System.Linq;
using System.Text.Json;
using CommunityToolkit.Mvvm.ComponentModel;
using CommunityToolkit.Mvvm.Input;
using Microsoft.Extensions.Logging;
using RobotgoFlow.Core.Models;
using RobotgoFlow.Core.Services;
using RobotgoFlow.Tray.Services;
using WpfApplication = System.Windows.Application;

namespace RobotgoFlow.Tray.ViewModels;

public partial class MiniPanelViewModel : ObservableObject
{
    private const int MaxRecentFiles = 10;
    private static readonly string RecentDir =
        Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), "RobotgoFlow");
    private static readonly string RecentFile = Path.Combine(RecentDir, "recent.json");

    private readonly ILogger _logger;
    private readonly IFileDialogService _fileDialog;

    [ObservableProperty] private string _workflowPath = "";
    [ObservableProperty] private string _statusText = "选择工作流文件后开始";
    [ObservableProperty] private bool _hasInputs;
    [ObservableProperty] private ObservableCollection<string> _recentFiles = new();
    [ObservableProperty] private ObservableCollection<InputField> _inputFields = new();

    public event Action<string, string, int, string?>? OnStartRequested;

    private string? _loadedName;

    public MiniPanelViewModel(ILogger logger, IFileDialogService fileDialog)
    {
        _logger = logger;
        _fileDialog = fileDialog;
        LoadRecentFiles();
    }

    public void LoadRecentFiles()
    {
        RecentFiles.Clear();
        try
        {
            if (File.Exists(RecentFile))
            {
                var json = File.ReadAllText(RecentFile);
                var list = JsonSerializer.Deserialize<List<string>>(json);
                if (list != null)
                    foreach (var f in list)
                        RecentFiles.Add(f);
            }
        }
        catch (Exception ex) when (ex is IOException or JsonException or UnauthorizedAccessException)
        {
            // 最近文件读写出错时静默降级，不影响核心功能
        }
    }

    private void SaveRecentFiles()
    {
        try
        {
            Directory.CreateDirectory(RecentDir);
            File.WriteAllText(RecentFile,
                JsonSerializer.Serialize(RecentFiles.Take(MaxRecentFiles).ToList()));
        }
        catch (Exception ex) when (ex is IOException or JsonException or UnauthorizedAccessException)
        {
            // 最近文件读写出错时静默降级，不影响核心功能
        }
    }

    private void AddToRecent(string path)
    {
        RecentFiles.Remove(path);
        RecentFiles.Insert(0, path);
        while (RecentFiles.Count > MaxRecentFiles)
            RecentFiles.RemoveAt(RecentFiles.Count - 1);
        SaveRecentFiles();
    }

    [RelayCommand]
    private void SelectWorkflow()
    {
        var path = _fileDialog.ShowOpenFileDialog("YAML 工作流|*.yaml;*.yml|所有文件|*.*");
        if (path != null)
        {
            WorkflowPath = path;
        }
    }

    partial void OnWorkflowPathChanged(string value)
    {
        if (string.IsNullOrWhiteSpace(value) || !File.Exists(value))
        {
            StatusText = "文件不存在";
            HasInputs = false;
            InputFields.Clear();
            return;
        }

        // 在后台线程执行 P/Invoke 调用，避免阻塞 UI 线程
        Task.Run(() =>
        {
            try
            {
                var resultJson = RobotgoNative.Instance.Preload(value);
                using var doc = JsonDocument.Parse(resultJson);
                var root = doc.RootElement;

                if (!root.TryGetProperty("ok", out var ok) || !ok.GetBoolean())
                {
                    var errorText = root.TryGetProperty("error", out var err)
                        ? $"加载失败: {err.GetString()}"
                        : "加载失败";
                    WpfApplication.Current.Dispatcher.InvokeAsync(() =>
                    {
                        StatusText = errorText;
                        HasInputs = false;
                    });
                    return;
                }

                var loadedNameVal = root.GetProperty("name").GetString() ?? "未知";
                var totalSteps = root.GetProperty("total_steps").GetInt32();
                var newFields = new List<InputField>();

                if (root.TryGetProperty("inputs", out var inputsEl))
                {
                    var loadedInputs = JsonSerializer.Deserialize<List<InputInfo>>(inputsEl.GetRawText());
                    if (loadedInputs is { Count: > 0 })
                    {
                        foreach (var inp in loadedInputs)
                        {
                            newFields.Add(new InputField
                            {
                                Name = inp.Name,
                                Label = inp.Label,
                                Placeholder = inp.Placeholder ?? "",
                                Required = inp.Required,
                                Mask = inp.Mask
                            });
                        }
                    }
                }

                WpfApplication.Current.Dispatcher.InvokeAsync(() =>
                {
                    _loadedName = loadedNameVal;
                    StatusText = $"「{loadedNameVal}」— {totalSteps} 个步骤";
                    HasInputs = newFields.Count > 0;
                    InputFields.Clear();
                    foreach (var f in newFields)
                        InputFields.Add(f);
                });
            }
            catch (Exception ex)
            {
                _logger.LogError(ex, "Preload 失败");
                WpfApplication.Current.Dispatcher.InvokeAsync(() =>
                {
                    _loadedName = null;
                    StatusText = $"加载失败: {ex.Message}";
                    HasInputs = false;
                });
            }
        });
    }

    [RelayCommand]
    private void StartExecution()
    {
        if (string.IsNullOrWhiteSpace(WorkflowPath) || !File.Exists(WorkflowPath))
        {
            StatusText = "请先选择有效的工作流文件";
            return;
        }

        var values = new Dictionary<string, string>();
        foreach (var field in InputFields)
        {
            if (field.Required && string.IsNullOrWhiteSpace(field.Value))
            {
                field.HasError = true;
                field.ErrorText = "此项必填";
            }
            else
            {
                field.HasError = false;
                field.ErrorText = "";
                if (!string.IsNullOrEmpty(field.Value))
                    values[field.Name] = field.Value;
            }
        }

        if (InputFields.Any(f => f.HasError))
        {
            StatusText = "请填写所有必填项";
            return;
        }

        AddToRecent(WorkflowPath);
        var inputsJson = values.Count > 0
            ? JsonSerializer.Serialize(values)
            : null;

        OnStartRequested?.Invoke(WorkflowPath, _loadedName ?? "未知工作流", 1, inputsJson);
    }

    [RelayCommand]
    private void Cancel()
    {
        OnStartRequested?.Invoke("", "", 0, null);
    }
}

public partial class InputField : ObservableObject
{
    public string Name { get; set; } = "";
    public string Label { get; set; } = "";
    public string Placeholder { get; set; } = "";
    [ObservableProperty] private string _value = "";
    [ObservableProperty] private bool _hasError;
    [ObservableProperty] private string _errorText = "";
    public bool Required { get; set; }
    // 注意：Mask 在运行时不应变更，仅在工作流加载时设置。
    // 如需动态变更，应改为 [ObservableProperty] 并触发 UI 刷新。
    public bool Mask { get; set; }
}
