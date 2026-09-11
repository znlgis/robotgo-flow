using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Text.Json;
using System.Threading;
using RobotgoFlow.Core.Helpers;
using RobotgoFlow.Core.Models;

namespace RobotgoFlow.Core.Services;

public class GoProcessService : IEngineService
{
    public const string DefaultGoBinaryPath = "robotgo-flow.exe";
    private const int StopTimeoutMs = 3000;

    private readonly object _stdinLock = new();
    private Process? _process;
    private StreamWriter? _stdin;

    public bool IsRunning => _process is { HasExited: false };

    public string GoBinaryPath { get; set; } = DefaultGoBinaryPath;

    public void Dispose()
    {
        Stop();
    }

    public event Action<ServeEvent>? OnEvent;
    public event Action<string>? OnStderr;
    public event Action<int>? OnExited;

    public void Start(string workflowPath, int fromStep = 1, bool debug = false)
    {
        Stop();

        var args = $"serve {ArgumentEscaper.Quote(workflowPath)}";
        if (fromStep > 1) args += $" --from {fromStep}";
        if (debug) args += " --debug";

        _process = new Process
        {
            StartInfo = new ProcessStartInfo
            {
                FileName = GoBinaryPath,
                Arguments = args,
                RedirectStandardInput = true,
                RedirectStandardOutput = true,
                RedirectStandardError = true,
                UseShellExecute = false,
                CreateNoWindow = true
            },
            EnableRaisingEvents = true
        };

        _process.OutputDataReceived += (_, e) =>
        {
            if (string.IsNullOrEmpty(e.Data)) return;
            try
            {
                var evt = JsonSerializer.Deserialize<ServeEvent>(e.Data);
                if (evt is not null)
                    OnEvent?.Invoke(evt);
            }
            catch (JsonException)
            {
                /* 忽略 JSON 解析错误 */
            }
        };

        _process.ErrorDataReceived += (_, e) =>
        {
            if (!string.IsNullOrEmpty(e.Data))
                OnStderr?.Invoke(e.Data);
        };

        // 捕获局部引用，防止另一线程将 _process 置 null 导致竞态
        var proc = _process;
        proc.Exited += (_, _) => { OnExited?.Invoke(proc.ExitCode); };

        _process.Start();
        _stdin = _process.StandardInput;

        _process.BeginOutputReadLine();
        _process.BeginErrorReadLine();
    }

    public void SendCommand(ServeCommand cmd)
    {
        if (_process is not { HasExited: false }) return;
        var json = JsonSerializer.Serialize(cmd);
        // 加锁：停止与输入命令可能来自不同线程，StreamWriter 不是线程安全的
        lock (_stdinLock)
        {
            var stdin = _stdin;
            if (stdin is null) return;
            try
            {
                stdin.WriteLine(json);
                stdin.Flush();
            }
            catch (IOException)
            {
                // 子进程已退出或管道已关闭，忽略
            }
            catch (ObjectDisposedException)
            {
            }
        }
    }

    /// <summary>
    ///     指定输入变量执行工作流。子进程模式通过 Start 启动后，通过 stdin 发送 inputs 命令。
    /// </summary>
    public void ExecuteWithInputs(string workflowPath, int fromStep, bool debug, string? inputsJson)
    {
        Start(workflowPath, fromStep, debug);
        if (!string.IsNullOrEmpty(inputsJson))
        {
            var values = JsonSerializer.Deserialize<Dictionary<string, string>>(inputsJson!);
            if (values != null)
                SendCommand(new ServeCommand("set_inputs", values));
        }
    }

    public void Stop()
    {
        // 使用 Interlocked.Exchange 原子地获取并置空，防止并发 Stop 竞态
        var proc = Interlocked.Exchange(ref _process, null);
        if (proc is null) return;

        try
        {
            if (!proc.HasExited)
            {
                // 发送停止命令到子进程（用局部变量避免 _process 已被置空）
                lock (_stdinLock)
                {
                    try { proc.StandardInput.WriteLine("{\"type\":\"stop\"}"); }
                    catch { /* stdin 可能已关闭 */ }
                }
                if (!proc.WaitForExit(StopTimeoutMs)) proc.Kill();
            }
        }
        catch { /* 进程已退出 */ }
        finally
        {
            lock (_stdinLock)
            {
                try { _stdin?.Dispose(); } catch { }
                _stdin = null;
            }
            try { proc.Dispose(); } catch { }
        }
    }
}
