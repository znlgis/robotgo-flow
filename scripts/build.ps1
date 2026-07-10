#!/usr/bin/env pwsh
# robotgo-flow build script
# Usage: .\scripts\build.ps1 [-Output <path>] [-NoStrip] [-Dll]
param(
    [string]$Output = "robotgo-flow.exe",
    [switch]$NoStrip,
    [switch]$SkipVerify,
    [switch]$Dll
)

$ErrorActionPreference = "Stop"

if ($Dll) {
    if ($Output -eq "robotgo-flow.exe") { $Output = "robotgo-flow.dll" }
}

Push-Location (Join-Path $PSScriptRoot "..\src\go")

try {
    # -- Prerequisites --
    Write-Host "[1/3] Checking environment..." -ForegroundColor Cyan

    try { $null = go version 2>&1 } catch {
        Write-Host "ERROR: Go not found. Please install Go 1.26+" -ForegroundColor Red
        exit 1
    }
    $goVer = (go version) -replace 'go(\d+\.\d+).*','$1'
    if ([version]$goVer -lt [version]"1.26") {
        Write-Error "需要 Go 1.26+，当前版本: $goVer"
        exit 1
    }
    Write-Host ("  Go: " + $goVer) -ForegroundColor Green

    $gccCmd = Get-Command gcc -ErrorAction SilentlyContinue
    if (-not $gccCmd) {
        Write-Host "错误: 未找到 MinGW-w64 GCC，请安装后重试" -ForegroundColor Red
        exit 1
    }
    Write-Host "GCC 路径: $($gccCmd.Source)" -ForegroundColor Green
    $gccMachine = gcc -dumpmachine 2>&1
    if (-not ($gccMachine -match "x86_64")) {
        Write-Error "需要 x86_64 MinGW-w64 GCC，当前 GCC 目标: $gccMachine"
        exit 1
    }
    Write-Host "GCC 目标: $gccMachine" -ForegroundColor Green

    # go mod verify
    Write-Host "  Verifying Go modules..." -ForegroundColor DarkGray
    go mod verify
    if ($LASTEXITCODE -ne 0) {
        Write-Error "go mod verify 失败"
        exit 1
    }

    # -- Build --
    Write-Host ""
    if ($Dll) {
        Write-Host "[2/3] Building DLL (c-shared)..." -ForegroundColor Cyan
        Write-Host "  First DLL build compiles GLFW C source (~4 min)" -ForegroundColor Yellow
    } else {
        Write-Host "[2/3] Building..." -ForegroundColor Cyan
        if (-not $NoStrip) {
            Write-Host "  First build takes ~4 min (compiling GLFW C source)" -ForegroundColor Yellow
            Write-Host "  Subsequent builds take ~1 sec (cache hit)" -ForegroundColor Yellow
        }
    }

    $stopwatch = [System.Diagnostics.Stopwatch]::StartNew()

    if ($Dll) {
        $buildArgs = @("build", "-buildmode=c-shared")
        if (-not $NoStrip) {
            $buildArgs += "-ldflags=-s -w"
        }
        $buildArgs += @("-o", $Output, "./internal/ffi/")
    } else {
        $ldflags = "-extldflags '-static'"
        if (-not $NoStrip) {
            $ldflags = "-s -w " + $ldflags
        }
        $buildArgs = @("build", "-v")
        if ($ldflags) {
            $buildArgs += "-ldflags=" + $ldflags
        }
        $buildArgs += @("-o", $Output, "./cmd/robotgo-flow/")
    }

    & go $buildArgs
    if ($LASTEXITCODE -ne 0) {
        Write-Host ""
        Write-Host ("Build FAILED (exit: " + $LASTEXITCODE + ")") -ForegroundColor Red
        exit $LASTEXITCODE
    }

    $stopwatch.Stop()
    $elapsed = [math]::Round($stopwatch.Elapsed.TotalSeconds)

    # -- Verify static linking --
    if ((-not $SkipVerify) -and (-not $Dll)) {
        Write-Host ""
        Write-Host "  Verifying static linking..." -ForegroundColor DarkGray
        $objdump = Get-Command objdump -ErrorAction SilentlyContinue
        if (-not $objdump) {
            $objdump = Get-Command "C:\msys64\mingw64\bin\objdump.exe" -ErrorAction SilentlyContinue
        }
        if ($objdump) {
            $deps = & $objdump -p $Output | Select-String "DLL Name"
            $badDlls = $deps | Where-Object { $_ -match "(libgcc|libstdc\+\+|libwinpthread|libpng)" }
            if ($badDlls) {
                Write-Host "  错误: 二进制依赖以下 MinGW 运行时 DLL，静态链接未生效:" -ForegroundColor Red
                Write-Host $badDlls
                Pop-Location
                exit 1
            }
            Write-Host "  通过: 无 MinGW 运行时 DLL 依赖" -ForegroundColor Green
        } else {
            Write-Host "  警告: 未找到 objdump，跳过静态链接验证" -ForegroundColor Yellow
        }
    }

    # DLL additional verification
    if ($Dll -and (-not $SkipVerify)) {
        Write-Host ""
        Write-Host "  Verifying DLL..." -ForegroundColor DarkGray
        $objdump = Get-Command objdump -ErrorAction SilentlyContinue
        if (-not $objdump) {
            $objdump = Get-Command "C:\msys64\mingw64\bin\objdump.exe" -ErrorAction SilentlyContinue
        }
        if ($objdump) {
            $deps = & $objdump -p $Output | Select-String "DLL Name"
            $badDlls = $deps | Where-Object { $_ -match "(libgcc|libstdc\+\+|libwinpthread)" }
            if ($badDlls) {
                Write-Host "  警告: DLL 依赖以下 MinGW 运行时:" -ForegroundColor Yellow
                Write-Host $badDlls
            } else {
                Write-Host "  通过: 无 MinGW 运行时 DLL 依赖" -ForegroundColor Green
            }
        }
    }

    # -- Result --
    Write-Host ""
    if ($Dll) {
        Write-Host "[3/3] DLL build done" -ForegroundColor Cyan
    } else {
        Write-Host "[3/3] Done" -ForegroundColor Cyan
    }
    $sizeMB = [math]::Round(((Get-Item $Output).Length / 1MB), 1)
    Write-Host ("  Output: " + $Output + " (" + $sizeMB + " MB)") -ForegroundColor Green
    Write-Host ("  Time:   " + $elapsed + "s") -ForegroundColor Green

    if ($Dll) {
        $hOutput = [System.IO.Path]::ChangeExtension($Output, ".h")
        if (Test-Path $hOutput) {
            Write-Host ("  Header: " + $hOutput) -ForegroundColor DarkGray
        }
    }

    # -- Build C# projects --
    Write-Host ""
    Write-Host "构建 C# 项目..." -ForegroundColor Cyan

    Push-Location (Join-Path $PSScriptRoot "..\src\csharp")
    try {
        dotnet build RobotgoFlow.Wpf.sln -c Release
        if ($LASTEXITCODE -ne 0) {
            Write-Host "C# 构建失败" -ForegroundColor Red
            exit $LASTEXITCODE
        }
        Write-Host "  C# 项目构建完成" -ForegroundColor Green
    } finally {
        Pop-Location
    }

} finally {
    Pop-Location
}

