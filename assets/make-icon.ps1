# ============================================================
# make-icon.ps1 — 应用图标生成：assets/appicon.png → build 产物
#
# 源：assets/appicon.png（512x512，入 git）+ assets/appicon.svg（矢量源）
# 产物：build/appicon.png（wails 图标源）+ build/windows/icon.ico
#       （多尺寸 256/64/48/32/16，嵌入 exe 窗口图标——wails 不会自动
#       从 appicon.png 重新生成 icon.ico，必须手动跑本脚本）
#
# 用法：powershell -File assets/make-icon.ps1（在仓库根执行）
# ============================================================

$RepoRoot = Split-Path $PSScriptRoot -Parent
$AppIcon = Join-Path $RepoRoot "assets\appicon.png"
$BuildIcon = Join-Path $RepoRoot "build\appicon.png"
$BuildIco = Join-Path $RepoRoot "build\windows\icon.ico"

if (-not (Test-Path $AppIcon)) { Write-Error "源图标缺失：$AppIcon"; exit 1 }
if (-not (Test-Path (Split-Path $BuildIco -Parent))) { New-Item -ItemType Directory -Force -Path (Split-Path $BuildIco -Parent) | Out-Null }

# 1) 复制到 build/appicon.png（wails 图标源）
Copy-Item $AppIcon $BuildIcon -Force

# 2) 生成多尺寸 icon.ico（PNG 嵌入，Vista+ 支持）
Add-Type -AssemblyName System.Drawing
$src = [System.Drawing.Image]::FromFile($AppIcon)
$sizes = @(256, 64, 48, 32, 16)
$images = @()
foreach ($s in $sizes) {
    $bmp = New-Object System.Drawing.Bitmap($s, $s)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $g.DrawImage($src, 0, 0, $s, $s)
    $g.Dispose()
    $ms = New-Object System.IO.MemoryStream
    $bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
    $images += ,@($s, $ms.ToArray())
    $bmp.Dispose()
    $ms.Dispose()
}
$src.Dispose()
$out = New-Object System.IO.MemoryStream
$bw = New-Object System.IO.BinaryWriter($out)
$bw.Write([UInt16]0); $bw.Write([UInt16]1); $bw.Write([UInt16]$images.Count)
$offset = 6 + 16 * $images.Count
foreach ($img in $images) {
    $s = $img[0]; $data = $img[1]
    $bw.Write([Byte]$(if ($s -ge 256) { 0 } else { $s }))
    $bw.Write([Byte]$(if ($s -ge 256) { 0 } else { $s }))
    $bw.Write([Byte]0); $bw.Write([Byte]0)
    $bw.Write([UInt16]1); $bw.Write([UInt16]32)
    $bw.Write([UInt32]$data.Length); $bw.Write([UInt32]$offset)
    $offset += $data.Length
}
foreach ($img in $images) { $bw.Write($img[1]) }
[System.IO.File]::WriteAllBytes($BuildIco, $out.ToArray())
$bw.Dispose(); $out.Dispose()

Write-Output "图标产物已生成："
Write-Output "  $BuildIcon ($((Get-Item $BuildIcon).Length) bytes)"
Write-Output "  $BuildIco ($((Get-Item $BuildIco).Length) bytes)"
Write-Output "下一步：wails build 重新构建"
