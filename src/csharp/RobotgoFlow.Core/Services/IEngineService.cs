using System;
using RobotgoFlow.Core.Models;

namespace RobotgoFlow.Core.Services;

/// <summary>
///     Go 引擎服务统一接口。RobotgoNative（DLL 直调）和 GoProcessService（子进程）
///     两种实现共享此接口，使 ViewModel 层不感知底层通信机制。
/// </summary>
public interface IEngineService : IDisposable
{
    /// <summary>是否正在运行</summary>
    bool IsRunning { get; }

    /// <summary>Go 二进制路径或 DLL 路径</summary>
    string GoBinaryPath { get; set; }

    // ── 事件 ──────────────────────────────────────────────
    event Action<ServeEvent>? OnEvent;
    event Action<string>? OnStderr;
    event Action<int>? OnExited;

    // ── 执行 ──────────────────────────────────────────────
    void Start(string workflowPath, int fromStep = 1, bool debug = false);

    /// <summary>
    ///     指定输入变量值执行工作流。DLL 模式直接调用，子进程模式通过 Start + stdin 传递。
    /// </summary>
    void ExecuteWithInputs(string workflowPath, int fromStep, bool debug, string? inputsJson);

    void SendCommand(ServeCommand cmd);
    void Stop();
}
