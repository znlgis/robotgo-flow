# robotgo-flow

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat&logo=go)](https://go.dev/dl/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows-blue.svg)]()
[![.NET](https://img.shields.io/badge/.NET-10-512BD4?style=flat&logo=dotnet)](https://dotnet.microsoft.com/)

基于 [robotgo](https://github.com/go-vgo/robotgo) 实现的 Windows 桌面 RPA 自动化框架。通过 YAML 定义工作流，利用图像模板匹配定位屏幕 UI 元素，自动执行鼠标、键盘、浏览器等操作。

- **Go CLI** — 纯命令行工具（`run` / `record` / `capture` / `serve`），专注自动化逻辑
- **WPF 托盘应用** — 基于 .NET 10 WPF 的 Windows 托盘应用，通过 DLL 导入与 Go 引擎通信

## 核心特性

- **YAML 驱动** — 用声明式 YAML 文件描述自动化流程，无需编写 Go 代码
- **WPF 托盘应用** — 基于 .NET 10 WPF 的 Windows 托盘应用，实时进度监控与任务栏通知
- **交互式动作** — 运行时输入框（支持密码隐藏）、确认对话框、系统通知，可实现人工决策节点
- **运行时变量注入** — 支持 `$input.<name>` 占位符，运行时动态替换文本内容
- **图像模板匹配** — 通过预截取的 UI 元素截图定位目标，适应窗口位置变化；窗口内优先搜索，全屏回退
- **人类行为模拟** — 贝塞尔曲线鼠标轨迹、打字错误与修正、可变延迟、空闲抖动，降低被检测风险
- **交互式录制器** — CLI 逐步引导录制工作流，自动创建模板截图
- **容错与调试** — 每步自动截图、从指定步骤恢复执行、调试模式；支持 abort / skip / retry 三种错误处理策略
- **安全停止** — 执行中可随时取消（context 取消机制），保证协程安全退出
- **截图工具** — 交互式区域框选截图
- **GBK/UTF-8 自动编码** — Windows 中文环境下自动处理 GBK 路径和 YAML 文件编码

## 快速开始

### 环境要求

| 组件 | 版本要求 | 说明 |
|------|---------|------|
| Go | 1.26+ | [下载地址](https://go.dev/dl/) |
| GCC (MinGW-w64) | x86_64 | 通过 MSYS2 安装 |
| .NET | 10.0+ | WPF 托盘应用需要 |
| Windows | 10 / 11 | 当前仅支持 Windows 平台 |

### 安装 MSYS2 和 GCC

```powershell
# 使用 winget 安装 MSYS2
winget install MSYS2.MSYS2

# 在 MSYS2 终端中安装编译工具链
pacman -S mingw-w64-x86_64-gcc mingw-w64-x86_64-zlib
```

将 `C:\msys64\mingw64\bin` 添加到系统 PATH。

### 编译

> [!NOTE]
> **首次编译耗时约 4 分钟**（需编译 GLFW C 源码），后续增量编译仅需 ~1 秒。

**Go CLI (必需)**

```powershell
.\scripts\build.ps1            # 优化输出 (strip 调试信息)
.\scripts\build.ps1 -NoStrip   # 调试版本 (保留符号)
```

**WPF 托盘应用**

```powershell
# 构建 C# 解决方案
cd src\csharp
dotnet build RobotgoFlow.Wpf.sln -c Release

# 构建 Tray 项目
dotnet build RobotgoFlow.Tray/RobotgoFlow.Tray.csproj -c Release
```

**Go DLL（供 Tray 应用调用）**

```powershell
.\scripts\build.ps1 -Dll         # 构建 c-shared DLL
.\scripts\build.ps1 -Dll -NoStrip # 调试版 DLL
```

### 第一个工作流

1. **编写 YAML 工作流**:

```yaml
name: "示例：登录表单"
settings:
  element_timeout: 10
  on_error: abort       # abort / skip / retry
  human:
    enabled: false
inputs:
  - name: username
    label: "用户名"
    required: true
  - name: password
    label: "密码"
    required: true
    mask: true
steps:
  - name: "打开网站"
    actions:
      - open_url: "https://example.com/login"
  - name: "登录"
    actions:
      - wait: "templates/btn_login.png"
      - click: "templates/btn_login.png"
      - type:
          into: "templates/input_username.png"
          text: "$input.username"
      - type:
          into: "templates/input_password.png"
          text: "$input.password"
      - press: "enter"
      - wait: "templates/welcome.png"
```

2. **截取 UI 元素模板截图**（使用 `capture` 命令或 `record` 交互录制）

3. **运行**:

```bash
# CLI — 完整执行
./robotgo-flow.exe run workflow.yaml

# CLI — 从第 3 步开始（1-indexed）
./robotgo-flow.exe run workflow.yaml --from 3

# CLI — 调试模式（每步自动截图到 screenshots/）
./robotgo-flow.exe run workflow.yaml --debug

# WPF GUI — 启动桌面应用
# 直接运行 WPF 编译输出（robotgo-flow-tray.exe）
# 加载 YAML → 输入变量 → 执行 → 实时监控
```

## CLI 命令

```bash
robotgo-flow                       # 无参数 → 显示帮助
robotgo-flow run <工作流文件>       # 执行工作流
robotgo-flow record                # 交互式录制工作流
robotgo-flow capture [元素名]      # 交互式截取模板
robotgo-flow serve <工作流文件>    # JSON-Line 协议服务 (供 WPF GUI 调用)
```

### `run` — 执行工作流

```bash
robotgo-flow run <workflow.yaml> [flags]
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--from N` | 1（从头） | 从第 N 步开始执行（1-indexed） |
| `--debug` | false | 调试模式，每步自动保存截图到输出目录 |
| `--out dir` | 工作流所在目录 | 截图输出目录（默认 `screenshots/`） |

运行时变量（`inputs` 字段中定义）通过命令行交互输入。

### `record` — 交互式录制工作流

```bash
robotgo-flow record [flags]
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--out` | `workflow.yml` | 输出的 YAML 文件路径 |
| `--tpl-dir` | `./templates` | 模板截图保存目录 |

按提示逐步录制：输入工作流元信息 → 添加步骤 → 选择动作类型（支持 19 种）→ 交互式截取模板图片 → 保存 YAML。

### `capture` — 截取模板图片

```bash
robotgo-flow capture [元素名] [flags]
```

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--out-dir` | `templates` | 截图输出目录 |

交互模式：提示用户将鼠标移到目标区域的左上角和右下角，自动框选区域并保存截图。

### `serve` — JSON-Line 协议服务

```bash
robotgo-flow serve <workflow.yaml>
```

启动 JSON-Line 协议服务（通过 stdin/stdout），供 WPF 托盘应用调用。

## WPF 托盘应用

基于 .NET 10 WPF 的 Windows 托盘应用（`RobotgoFlow.Tray`），通过 DLL 导入直接调用 Go 引擎，提供任务栏快捷操作、实时进度浮窗和桌面通知。

> **注意**：WPF 托盘应用为可选组件。Go CLI 可独立使用，不依赖 .NET 或 WPF。

## YAML 工作流参考

### 顶层结构

```yaml
name: "工作流名称"              # 必填
description: "描述"              # 可选
inputs:                          # 可选，运行时变量定义
  - name: username               #   变量名（引用方式: $input.username）
    label: "用户名"              #   输入提示标签
    required: true               #   是否必填
    placeholder: "请输入用户名"  #   占位符文本（可选）
    mask: false                  #   是否隐藏输入（密码模式）
settings:                        # 可选，全局设置
  element_timeout: 10            #   等待元素出现的超时秒数（默认 10）
  on_error: abort                #   错误处理策略：abort / skip / retry（默认 abort）
  max_retries: 3                 #   retry 模式下的最大重试次数（默认 3）
  browser_refresh_delay: 3       #   刷新页面等待秒数（默认 3）
  browser_navigation_delay: 2    #   前进/后退等待秒数（默认 2）
  browser_page_load_delay: 3     #   打开 URL 等待秒数（默认 3）
  human:                         #   人类行为模拟设置
    enabled: false               #   是否启用
    speed: 1.0                   #   速度系数（0.1 ~ 5.0，默认 1.0）
    mistake_rate: 0.03           #   打字错误率（0.0 ~ 1.0，默认 0.0）
steps:                           # 必填，步骤列表
  - name: "步骤名"
    actions: [...]               #   动作列表
```

### `inputs` — 运行时变量

工作流中的文本内容可以使用 `$input.<变量名>` 占位符。执行前会提示用户输入对应变量的值：

```yaml
inputs:
  - name: username
    label: "用户名"
    required: true
  - name: password
    label: "密码"
    required: true
    mask: true      # 密码输入（隐藏显示）
  - name: comment
    label: "备注"
    placeholder: "可选填写"
```

在动作中使用：

```yaml
- type: {into: "templates/input.png", text: "$input.username"}
```

### `on_error` — 错误处理

| 值 | 说明 |
|----|------|
| `abort` | 立即终止执行（默认） |
| `skip` | 记录错误日志，跳过当前动作，继续执行下一个 |
| `retry` | 重试当前动作，最多 `max_retries` 次；耗尽后 abort |

### 动作类型（19 种）

#### 鼠标操作

```yaml
# 单击 — 模板匹配或坐标
- click: "templates/button.png"
- click: {x: 500, y: 300}

# 双击
- double_click: "templates/item.png"
- double_click: {x: 500, y: 300}

# 右键
- right_click: "templates/menu.png"
- right_click: {x: 500, y: 300}

# 拖拽（从模板 A 到模板 B）
- drag: {from: "templates/source.png", to: "templates/target.png"}
```

#### 键盘操作

```yaml
# 在输入框中输入文本
- type: {into: "templates/input.png", text: "Hello World"}

# 按下单个键（支持特殊键名：enter / tab / escape / backspace / space 等）
- press: "enter"

# 组合键
- combo: ["ctrl", "c"]
```

#### 等待与延时

```yaml
# 等待元素出现（默认超时为 settings.element_timeout）
- wait: "templates/loading_done.png"
- wait: {template: "templates/popup.png", timeout: 30}

# 等待元素消失
- wait_gone: "templates/loading.png"

# 固定延时（秒，支持小数）
- sleep: 2.5
```

#### 浏览器操作

```yaml
# 打开 URL（通过剪贴板 + 地址栏输入，自动定位浏览器窗口）
- open_url: "https://example.com"

# 刷新页面（Ctrl+R）
- refresh: true

# 后退（Alt+Left）
- back: true

# 前进（Alt+Right）
- forward: true

# 切换标签页（1-9，Ctrl+数字）
- switch_tab: 2
```

#### 滚动

```yaml
# 正数向下、负数向上
- scroll: 500
- scroll: -300
```

#### 交互式动作（需要 GUI 或控制台交互）

```yaml
# 运行时输入提示 — 让用户输入内容，支持密码模式
- prompt:
    title: "请输入验证码"
    message: "验证码已发送至手机"
    into: "templates/input_code.png"   # 可选：输入前先点击目标模板
    mask: false

# 确认对话框 — 让用户确认是否继续
- confirm:
    title: "确认操作"
    message: "数据已填写完毕，是否提交？"

# 系统通知
- notify:
    title: "提醒"
    message: "处理已完成"
    duration: 3   # 秒，>0 时自动消失；=0 时需手动关闭
```

### 完整示例

参见上方 YAML 工作流参考中的完整示例。

## 图像模板匹配

### 工作原理

1. 用户预先截取目标 UI 元素的截图（PNG 格式），保存到 `templates/` 目录
2. 工作流运行时，工具通过 `robotgo.FindBitmap()` 在屏幕上搜索这些模板
3. 定位成功后，在匹配位置中心执行鼠标 / 键盘操作

### 搜索策略

- **窗口内搜索（优先）**：如果检测到浏览器窗口（Chrome / Edge / Firefox / Brave / Opera），截取窗口区域进行图像匹配，避免匹配到浏览器外的无关内容
- **全屏回退**：窗口内未找到时，回退到全屏搜索

工作流执行开始时自动激活并最大化浏览器窗口，`open_url` 操作后也会重新定位窗口确保匹配区域准确。

### 创建模板截图

三种方式：

1. **WPF 录制标签页** — 一键启动交互式录制器
2. **`capture` 命令** — 交互式框选屏幕区域
3. **`CaptureInteractive` API** — 编程调用交互式截图（`capture.CaptureInteractive` / `capture.CaptureInteractivePipe`），支持 stdin 和管道两种模式

### 模板路径

YAML 中引用的模板路径相对于工作流 YAML 文件所在目录。推荐约定：

```
project/
├── workflow.yaml
└── templates/
    ├── btn_login.png
    ├── input_username.png
    └── ...
```

## 人类行为模拟

通过 YAML 中 `settings.human.enabled: true` 启用。模拟内容包括：

| 行为 | 说明 |
|------|------|
| 贝塞尔曲线鼠标轨迹 | 鼠标移动使用自然曲线路径，靠近目标时减速 |
| 打字错误模拟 | 基于 `mistake_rate` 引入相邻键错误、漏字、顺序颠倒，并自动修正 |
| 可变延迟 | 每次操作间的间隔时间在随机范围内变化 |
| 滚动抖动 | 滚动时逐步分块执行，有一定概率轻微反向回滚 |
| 空闲行为 | 长时间等待时偶有鼠标微抖动；步骤间有概率移动鼠标到随机位置 |

```yaml
settings:
  human:
    enabled: true
    speed: 0.8         # 0.8x 速度（比正常慢）
    mistake_rate: 0.03 # 3% 字符打错率
```

## 编码处理

Windows 中文环境下，PowerShell 默认编码为 GBK，而 Go/robotgo 底层使用 UTF-8。`robotgo-flow` 通过 `internal/encoding` 包自动处理编码转换：

- **CLI 参数**：`os.Args` 中的中文自动从 GBK 转为 UTF-8
- **YAML 文件**：加载时自动检测并转换 GBK → UTF-8
- **模板路径**：传递给 CGo 层（robotgo bitmap 保存 API）的文件路径转为 GBK

用户无需手动处理编码问题。

## 项目结构

```
robotgo-flow/
├── src/
│   ├── go/
│   │   ├── cmd/robotgo-flow/
│   │   │   └── main.go                  # 入口：CLI 子命令（run / record / capture / serve）
│   │   ├── internal/
│   │   │   ├── config/
│   │   │   │   ├── workflow.go          # YAML 数据结构定义（Workflow, Step, Action, InputSpec 等）
│   │   │   │   ├── loader.go            # YAML 加载、解析、多层验证、默认值填充
│   │   │   │   ├── loader_test.go       # 加载器单元测试
│   │   │   │   └── loader_ext_test.go   # 扩展测试（外部测试包）
│   │   │   ├── engine/
│   │   │   │   ├── engine.go            # 核心引擎：封装 robotgo（鼠标、键盘、滚动、截图、浏览器管理、模板缓存）
│   │   │   │   ├── human.go             # 人类行为模拟（贝塞尔曲线、打字错误、空闲行为）
│   │   │   │   ├── matcher.go           # 图像匹配（模板查找、窗口内搜索、等待元素、容差匹配）
│   │   │   │   └── keyboard_layout.go   # QWERTY 键盘邻接映射（用于打字错误模拟）
│   │   │   ├── action/
│   │   │   │   ├── action.go            # Runner 接口与 Engine 接口定义
│   │   │   │   ├── factory.go           # FromConfig 工厂函数（config.Action → Runner 实现）
│   │   │   │   ├── factory_test.go      # 工厂函数测试
│   │   │   │   ├── execute_test.go      # 动作执行集成测试
│   │   │   │   ├── interact.go          # 交互式动作（prompt / confirm / notify）
│   │   │   │   ├── interact_test.go     # 交互动作测试
│   │   │   │   ├── engine_mock.go       # Engine 接口 Mock（单元测试用）
│   │   │   │   ├── click.go             # 单击 / 双击 / 右键
│   │   │   │   ├── drag.go              # 拖拽
│   │   │   │   ├── key.go               # 按键 / 组合键
│   │   │   │   ├── type.go              # 输入文本
│   │   │   │   ├── wait.go              # 等待元素出现 / 消失
│   │   │   │   ├── sleep.go             # 固定延时
│   │   │   │   ├── scroll.go            # 鼠标滚动
│   │   │   │   └── navigate.go          # 浏览器操作（open_url / refresh / back / forward / switch_tab）
│   │   │   ├── executor/
│   │   │   │   └── executor.go          # 工作流执行器：遍历步骤，调用 action 执行；支持 runtime input 解析、context 取消
│   │   │   ├── serve/
│   │   │   │   └── serve.go             # JSON-Line 协议服务（供 WPF GUI 调用）
│   │   │   ├── notify/
│   │   │   │   └── notify.go            # 标准输入/输出通知（输入框、确认框、错误日志）
│   │   │   ├── recorder/
│   │   │   │   └── recorder.go          # 交互式录制器：CLI 引导录制 18 种动作类型
│   │   │   ├── capture/
│   │   │   │   └── capture.go           # 截图工具：交互式框选截图（CaptureRegion / CaptureInteractive / CaptureInteractivePipe）
│   │   │   ├── geom/
│   │   │   │   └── geom.go              # 屏幕几何类型（Point）
│   │   │   └── encoding/
│   │   │       └── encoding.go          # Windows GBK ↔ UTF-8 编码转换
│   │   ├── go.mod
│   │   └── go.sum
│   └── csharp/
│       ├── RobotgoFlow.Core/            # 共享核心库 (.NET 10 / .NET Framework 4.8 双目标)
│       │   ├── Models/
│       │   │   └── ProtocolMessages.cs  # JSON-Line 协议消息类型
│       │   ├── Services/
│       │   │   ├── GoProcessService.cs  # Go 子进程管理
│       │   │   ├── IEngineService.cs    # 引擎服务接口
│       │   │   └── RobotgoNative.cs     # P/Invoke 原生方法声明
│       │   └── Helpers/
│       │       └── ArgumentEscaper.cs   # 命令行参数转义
│       ├── RobotgoFlow.Tray/            # WPF 托盘应用 (.NET 10)
│       │   ├── App.xaml                 # 应用入口
│       │   ├── MiniPanelWindow.xaml     # 迷你面板窗口
│       │   ├── ProgressOverlay.xaml     # 进度浮窗
│       │   ├── TrayIcon.cs              # 系统托盘图标
│       │   ├── ToastNotifier.cs         # 桌面通知
│       │   ├── Services/
│       │   │   ├── ExecutionService.cs    # 工作流执行服务
│       │   │   └── IFileDialogService.cs  # 文件对话框服务（接口与实现）
│       │   └── ViewModels/
│       │       ├── MiniPanelViewModel.cs
│       │       └── ProgressViewModel.cs
│       └── RobotgoFlow.Wpf.sln         # C# 解决方案
└── scripts/
    └── build.ps1                # Go 构建脚本
```

### 架构关系

```
cmd/robotgo-flow/main.go
    │
    ├── config.Load()              → 解析 YAML → *Workflow
    ├── engine.NewEngine()         → 封装 robotgo → *Engine (含模板缓存、浏览器窗口定位)
    ├── executor.New()             → 绑定 Engine + Workflow → *Executor
    │   ├── executor.ResolveInputs()  → 解析 $input.<name> 占位符
    │   ├── executor.WithContext()    → 注入 context 支持取消
    │   └── executor.Run()         → 遍历步骤 → action.FromConfig() → act.Execute(eng)
    │
    ├── serve.Run()                → JSON-Line 协议服务 → 供 WPF GUI 调用
    │   └── executor.ProgressCallback → 实时进度事件流
    │
    ├── recorder.New()             → 交互式录制 → 生成 YAML + 模板截图
    └── capture.Capture*()         → 独立截图工具（区域截图 / 交互式截图）

WPF 托盘应用 (C#, .NET 10)
    │
    ├── GoProcessService           → 启动 robotgo-flow serve 子进程
    ├── JSON-Line stdin/stdout     → 双向通信协议
    └── ExecutionViewModel         → 进度条、日志、输入变量采集
```

`internal/action` 定义了 `Runner` 和 `Engine` 两个接口，作为引擎与执行器之间的契约，确保关注点清晰分离。

## 测试

```bash
# 进入 Go 源码目录
cd src/go

# 运行所有测试
go test ./...

# 运行指定包的测试
go test ./internal/config/ -v
go test ./internal/action/ -v
go test ./internal/executor/ -v

# 带覆盖率
go test ./... -cover
```

## 扩展开发

### 添加新的动作类型

1. 在 `src/go/internal/config/workflow.go` 的 `Action` 结构体中添加新字段，并更新 `ActionType()` 和 `Label()` 方法
2. 在 `src/go/internal/action/` 下新建文件，实现 `Runner` 接口（`Execute` 方法）
3. 在 `src/go/internal/action/factory.go` 的 `FromConfig()` 中添加对应分支
4. 在 `src/go/internal/config/loader.go` 的 `isEmpty()` 和 `validateSingleAction()` 中添加新类型的处理
5. 在 `src/go/internal/recorder/recorder.go` 中添加录制支持

### 适配其他平台

- 图像模板匹配和鼠标 / 键盘操作依赖 robotgo，理论支持 Linux / macOS
- 浏览器窗口检测使用了 Windows 特有 API（`robotgo.FindIds`、`robotgo.ActivePid`），需适配对应平台 API
- 组合键使用了 `alt` / `ctrl`，跨平台可能需要调整修饰键名称

## 故障排查

| 问题 | 可能原因 | 解决方法 |
|------|---------|---------|
| 模板匹配失败 | 屏幕分辨率变化 / DPI 缩放不同 / UI 内容变化 | 重新截取模板，确保分辨率一致；关闭 Windows 显示缩放 |
| 浏览器窗口未检测到 | 浏览器未打开 / 窗口最小化 / 多进程架构干扰 | 确保浏览器窗口可见；检查是否被其他窗口遮挡 |
| 鼠标点击位置偏移 | 窗口未最大化 / 标题栏占用高度 | 工具会自动最大化窗口；检查截图时和运行时的窗口状态一致 |
| 编译错误 | GCC 未安装或未添加到 PATH | 确认 `gcc --version` 可用；检查 `C:\msys64\mingw64\bin` 在 PATH 中 |
| YAML 解析失败 | 文件为 GBK 编码 | 工具自动转换，无需手动处理；若仍有问题，将 YAML 另存为 UTF-8 |
| `capture` 截图黑屏 | 目标窗口被遮挡 / DPI 虚拟化 | 确保目标区域可见；以管理员权限运行 |

## 注意事项

- **屏幕分辨率**：模板截图应与运行环境分辨率一致，否则可能匹配失败。建议在目标机器上截取模板，而非异地截图
- **窗口状态**：运行前确保浏览器窗口可见且未被遮挡；工具会尝试自动激活并最大化窗口
- **模板质量**：截取模板时避免包含可变内容（如文本、时间戳），选择稳定的 UI 区域（图标、按钮边缘等）
- **安全性**：工作流 YAML 中不要包含密码等敏感信息；对于密码类变量，推荐使用运行时 `inputs` 的 `mask: true` 模式或 `prompt` 动作的 `mask` 参数

## License

[MIT](LICENSE)
