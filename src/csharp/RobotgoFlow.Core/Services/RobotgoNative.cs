using System;
using System.Diagnostics;
using System.Runtime.InteropServices;
using System.Text;
using System.Text.Json;
using System.Threading;
using System.Threading.Tasks;
using RobotgoFlow.Core.Models;

namespace RobotgoFlow.Core.Services;

/// <summary>
///     Go 引擎 DLL 直调实现 — 通过 P/Invoke 调用 robotgo-flow.dll。
///     单例模式：DLL 全局状态（回调/执行器）不支持多实例。
/// </summary>
public class RobotgoNative : IEngineService
{
    // ── P/Invoke 声明 ────────────────────────────────────────

    private const string DllName = "robotgo-flow.dll";

    // 静态持有，防止 GC 回收（Go 侧持有此函数指针）
    private static readonly RawCallbackDelegate? _staticCallback;

    // ── JSON 序列化选项（与 Go snake_case 输出匹配） ──────────

    private static readonly JsonSerializerOptions _jsonOptions = new()
    {
        PropertyNameCaseInsensitive = true,
        PropertyNamingPolicy = JsonNamingPolicy.SnakeCaseLower
    };

    // ── 实例状态 ──────────────────────────────────────────────

    private Task? _executeTask;
    private int _isDisposed;
    private volatile bool _isRunning;

    static RobotgoNative()
    {
        try
        {
            _staticCallback = OnRawCallback; // IntPtr → UTF-8 → string → OnGoCallback
            RobotgoSetCallback(_staticCallback);
            var verPtr = RobotgoInit();
            var ver = MarshalNativeString(verPtr);
            Debug.WriteLine($"[RobotgoNative] DLL 已加载: {ver}");
            IsAvailable = true;
        }
        catch (DllNotFoundException ex)
        {
            Debug.WriteLine($"[RobotgoNative] DLL 未找到: {ex.Message}");
            IsAvailable = false;
        }
        catch (Exception ex)
        {
            Debug.WriteLine($"[RobotgoNative] DLL 初始化失败: {ex.Message}");
            IsAvailable = false;
        }
    }

    private RobotgoNative()
    {
        GoBinaryPath = "robotgo-flow.dll";
    }
    // ── 单例 ──────────────────────────────────────────────────

    public static RobotgoNative Instance { get; } = new();

    /// <summary>DLL 是否成功加载并初始化</summary>
    public static bool IsAvailable { get; private set; }

    /// <summary>
    ///     回调调度器。WPF 设置此委托以将事件调度到 UI 线程；
    ///     控制台程序不设置，事件直接在回调线程上同步执行。
    /// </summary>
    public static Action<Action>? DispatchCallback { get; set; }

    // ── IEngineService ─────────────────────────────────────────

    public bool IsRunning => _isRunning;
    public string GoBinaryPath { get; set; }

    public event Action<ServeEvent>? OnEvent;
    public event Action<string>? OnStderr;
    public event Action<int>? OnExited;

    public void Start(string workflowPath, int fromStep = 1, bool debug = false)
    {
        ExecuteWithInputs(workflowPath, fromStep, debug, null);
    }

    public void SendCommand(ServeCommand cmd)
    {
        // DLL 模式下仅支持 stop 命令。set_inputs 应通过 Start 的 inputsJson 参数传递。
        if (cmd.Type == "stop")
            RobotgoStop();
        else
            Debug.WriteLine($"[RobotgoNative] 忽略命令: {cmd.Type}");
    }

    public void Stop()
    {
        RobotgoStop();
        _isRunning = false;
    }

    // ── 清理 ──────────────────────────────────────────────────

    public void Dispose()
    {
        if (Interlocked.Exchange(ref _isDisposed, 1) != 0) return;
        Stop();
        // 等待执行任务完成，避免 use-after-free
        try { _executeTask?.Wait(TimeSpan.FromSeconds(5)); }
        catch { /* 超时或异常，继续销毁 */ }
        RobotgoDestroy();
        IsAvailable = false;
    }

    [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
    private static extern IntPtr RobotgoInit();

    [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
    private static extern void RobotgoDestroy();

    [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
    private static extern void RobotgoSetCallback(RawCallbackDelegate cb);

    [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
    private static extern void RobotgoFreeString(IntPtr str);

    [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
    private static extern IntPtr RobotgoExecute(
        [MarshalAs(UnmanagedType.LPStr)] string workflowPath,
        int fromStep,
        int debug,
        [MarshalAs(UnmanagedType.LPStr)] string? inputsJson);

    [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
    private static extern void RobotgoStop();

    [DllImport(DllName, CallingConvention = CallingConvention.Cdecl)]
    private static extern IntPtr RobotgoPreload(
        [MarshalAs(UnmanagedType.LPStr)] string workflowPath);

    /// <summary>
    ///     预加载工作流配置，返回元数据 JSON（name, total_steps, inputs）。
    ///     不启动执行，用于 DLL 模式下在执行前采集输入变量。
    /// </summary>
    public string Preload(string workflowPath)
    {
        ThrowIfDisposed();
        var ptr = RobotgoPreload(workflowPath);
        return MarshalNativeString(ptr);
    }

    /// <summary>
    ///     指定输入变量值执行工作流（DLL 模式专用）。
    ///     必须在 Start/Preload 之前采集好输入变量。
    /// </summary>
    public void ExecuteWithInputs(string workflowPath, int fromStep, bool debug, string? inputsJson)
    {
        ThrowIfDisposed();
        if (_isRunning) return;
        _isRunning = true;

        _executeTask = Task.Run(() =>
        {
            try
            {
                var ptr = RobotgoExecute(workflowPath, fromStep, debug ? 1 : 0, inputsJson);
                var result = MarshalNativeString(ptr);

                // 使用 JsonDocument 解析替代字符串匹配，更可靠地检测错误
                if (!string.IsNullOrEmpty(result))
                {
                    try
                    {
                        using var doc = JsonDocument.Parse(result);
                        if (doc.RootElement.TryGetProperty("ok", out var ok) && ok.GetBoolean() == false)
                            Debug.WriteLine($"[RobotgoNative] 执行失败: {result}");
                    }
                    catch { /* JSON 解析失败，忽略 */ }
                }
            }
            catch (Exception ex)
            {
                Debug.WriteLine($"[RobotgoNative] 执行异常: {ex}");
            }
            finally
            {
                _isRunning = false;
                var d = DispatchCallback;
                if (d != null)
                    d(() => OnExited?.Invoke(0));
                else
                    OnExited?.Invoke(0);
            }
        });
    }

    private void ThrowIfDisposed()
    {
        if (_isDisposed != 0)
            throw new ObjectDisposedException(nameof(RobotgoNative));
    }

    // ── 回调处理 ──────────────────────────────────────────────

    /// <summary>接收原始指针，手动按 UTF-8 解码后转发给 OnGoCallback。</summary>
    private static void OnRawCallback(IntPtr jsonPtr)
    {
        var json = PtrToStringUTF8(jsonPtr);
        if (json == null) return;
        OnGoCallback(json);
    }

    private static void OnGoCallback(string json)
    {
        var dispatcher = DispatchCallback;
        if (dispatcher != null)
            dispatcher(() => ProcessEvent(json));
        else
            ProcessEvent(json);
    }

    private static void ProcessEvent(string json)
    {
        if (Instance._isDisposed != 0) return;
        try
        {
            var evt = JsonSerializer.Deserialize<ServeEvent>(json, _jsonOptions);
            if (evt == null) return;
            Instance.OnEvent?.Invoke(evt);
        }
        catch (JsonException ex)
        {
            Debug.WriteLine($"[RobotgoNative] 事件解析失败: {ex.Message}");
        }
    }

    // ── 内存管理 ──────────────────────────────────────────────

    /// <summary>
    ///     从 Go 返回的 IntPtr 读取 UTF-8 字符串并释放内存。
    ///     Go 侧 C.CString 输出 UTF-8 字节；.NET Framework 4.8 无
    ///     Marshal.PtrToStringUTF8，手动实现。
    /// </summary>
    private static string MarshalNativeString(IntPtr ptr)
    {
        if (ptr == IntPtr.Zero) return string.Empty;
        try
        {
            return PtrToStringUTF8(ptr);
        }
        finally
        {
            RobotgoFreeString(ptr);
        }
    }

    /// <summary>
    ///     从 IntPtr 读取以 \0 结尾的 UTF-8 字节并解码为 string。
    /// </summary>
    private static string PtrToStringUTF8(IntPtr ptr)
    {
        if (ptr == IntPtr.Zero) return string.Empty;
        var len = 0;
        while (Marshal.ReadByte(ptr, len) != 0) len++;
        if (len == 0) return string.Empty;
        var bytes = new byte[len];
        Marshal.Copy(ptr, bytes, 0, len);
        return Encoding.UTF8.GetString(bytes);
    }

    // ── 委托 ──────────────────────────────────────────────────

    /// <summary>
    ///     本地回调：接收原始指针，手动按 UTF-8 解码（.NET Framework 4.8
    ///     不支持 Marshal.PtrToStringUTF8，LPStr 会将 UTF-8 字节误按 ANSI/GBK 解码）。
    /// </summary>
    [UnmanagedFunctionPointer(CallingConvention.Cdecl)]
    private delegate void RawCallbackDelegate(IntPtr jsonPtr);
}
