# AGENTS.md — robotgo-flow

## 项目概述

Windows 桌面 RPA 自动化框架。Go CLI（核心引擎）+ .NET 10 WPF 托盘应用（桌面界面），通过 DLL 导入直接调用 Go 引擎。

## 工作目录

所有 Go 命令必须在 **`src/go/`** 下执行：

```powershell
cd src/go
go build ./...
go test ./... -count=1
```

## 构建与运行

```powershell
# 构建 Go CLI（首次 ~4min，后续 ~1s；需要 MinGW-w64 GCC）
.\scripts\build.ps1

# 运行单个测试包
go test -v ./internal/config/ -run TestLoad

# 运行基准测试
go test -bench . -benchtime 1x ./internal/encoding/
```

### 环境前提

- **Go 1.26+** 且 `CGO_ENABLED=1`（默认）
- **MinGW-w64 GCC (x86_64)**：`C:\msys64\mingw64\bin` 必须在 PATH 中
- **.NET 10 SDK**（构建 C# 项目需要）
- **Windows 10/11**（robotgo 仅支持 Windows）

### 构建 C# 项目

```powershell
# 构建 C# 解决方案
cd src\csharp
dotnet build RobotgoFlow.Wpf.sln -c Release

# 构建 Tray 项目
dotnet build RobotgoFlow.Tray/RobotgoFlow.Tray.csproj -c Release

# 构建 Core 库
dotnet build RobotgoFlow.Core/RobotgoFlow.Core.csproj -c Release
```

## 项目结构（关键路径）

```
src/
├── go/
│   ├── cmd/robotgo-flow/main.go     # CLI 入口（run/record/capture/serve/version）
│   └── internal/
│       ├── action/      # 19 种动作实现（Runner 接口）+ MockEngine
│       ├── engine/      # robotgo 封装（鼠标/键盘/图像匹配/人类模拟）
│       ├── config/      # YAML 加载/校验，Workflow/Step/Action 结构体
│       ├── executor/    # 工作流执行器（ProgressCallback，输入占位符，context 取消）
│       ├── serve/       # JSON-Line 协议（供 WPF GUI 调用）
│       ├── recorder/    # CLI 交互式录制
│       ├── capture/     # 交互式区域截图
│       ├── notify/      # stdin/stdout 弹窗（输入框/确认框/错误日志）
│       ├── logger/      # 统一日志（Info/Warn/Error/Debug 四级）
│       ├── encoding/    # GBK ↔ UTF-8 转换
│       └── geom/        # Point 类型
├── csharp/
│   ├── RobotgoFlow.Core/           # 共享核心库 (net48 + net10.0 双目标)
│   │   ├── Models/ProtocolMessages.cs   # JSON-Line 协议消息类型
│   │   ├── Services/GoProcessService.cs # Go 子进程管理
│   │   ├── Services/IEngineService.cs   # 引擎服务接口
│   │   ├── Services/RobotgoNative.cs    # P/Invoke 原生方法
│   │   └── Helpers/ArgumentEscaper.cs   # 命令行参数转义
│   ├── RobotgoFlow.Tray/           # WPF 托盘应用 (.NET 10)
│   │   ├── App.xaml                     # 应用入口
│   │   ├── MiniPanelWindow.xaml         # 迷你面板窗口
│   │   ├── ProgressOverlay.xaml         # 进度浮窗
│   │   ├── TrayIcon.cs                  # 系统托盘图标
│   │   ├── ToastNotifier.cs             # 桌面通知
│   │   ├── Services/ExecutionService.cs # 工作流执行服务
│   │   ├── Services/IFileDialogService.cs # 文件对话框服务
│   │   └── ViewModels/
│   │       ├── MiniPanelViewModel.cs
│   │       └── ProgressViewModel.cs
│   └── RobotgoFlow.Wpf.sln         # C# 解决方案
```

## 架构约定

1. **接口分离**：`action.Runner`（动作执行）和 `action.Engine`（底层能力）。测试全部通过 `MockEngine` 完成。
2. **配置验证**：多层校验 — 字段完整性 → 模板文件存在性 → 单动作约束 → 默认值填充。
3. **图像匹配**：优先在浏览器窗口区域内搜索，失败后回退到全屏搜索。模板位图缓存在 Engine 中，由 `Close()` 统一释放。
4. **错误处理**：`ProgressCallback.OnStepError` 返回 abort/skip/retry 三种策略。执行通过 `context.Context` 支持取消。
5. **人类模拟**：贝塞尔曲线鼠标路径、QWERTY 邻接打字错误、空闲抖动、滚动噪声。

## 语言约定

**全部使用简体中文**：代码注释、错误消息（`fmt.Errorf`）、日志输出（`logger.*`）、界面字符串（XAML + C# ViewModel）均以简体中文书写。不保留英文注释或繁体中文。

## 依赖约束

- 仅使用 `go.mod` 中已有的依赖；不要引入新的第三方库
- CGo 库（robotgo, bitmap）是必需依赖，不能用纯 Go 替代
- 不要引入新的 NuGet 包或 .NET 库（RobotgoFlow.Tray 项目已有的 NuGet 包除外）

## CI/CD

Windows-only CI（`.github/workflows/test.yml`），步骤：`go mod verify` → `go vet` → `go build` → `go test -count=1 -timeout=5m` → `dotnet build`。

## 已知问题

- `engine.go`：`openCachedTemplate` 通过 `templateCache` map + `sync.RWMutex` 缓存模板位图，`Engine.Close()` 统一释放。`FindElement` / `findInWindow` 均通过该方法获取模板，避免重复 `bitmaputil.Open`
- 首次 `go build` 需编译 GLFW C 源码，耗时约 4 分钟；后续增量编译秒级完成
