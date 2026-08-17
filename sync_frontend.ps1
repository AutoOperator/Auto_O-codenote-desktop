# ============================================================
# sync_frontend.ps1 — 前端 A 同步：code-note-obsidian → Auto_O-codenote-desktop/frontend
#
# A（笔记主页.html + static/）单一来源为 code-note-obsidian 仓库，
# 本脚本幂等复制到桌面壳（SHA256 哈希比对，未变更跳过）。
# frontend/obsidian-bridge.js（C 包装）由本仓库维护，不覆盖。
#
# 用法：
#   powershell -File sync_frontend.ps1            # 执行同步
#   powershell -File sync_frontend.ps1 -Verify    # 只比对报告，不复制
# ============================================================

param(
    [switch]$Verify
)

$RepoRoot = $PSScriptRoot
$SrcRoot = Join-Path (Split-Path $RepoRoot -Parent) "code-note-obsidian"
$DstRoot = Join-Path $RepoRoot "frontend"

if (-not (Test-Path (Join-Path $SrcRoot "笔记主页.html"))) {
    Write-Error "源 A 缺失（未找到 code-note-obsidian\笔记主页.html）：$SrcRoot"
    exit 1
}
if (-not (Test-Path $DstRoot)) { New-Item -ItemType Directory -Force -Path $DstRoot | Out-Null }

$copied = @()
$unchanged = @()
$missing = @()

# 主文件：笔记主页.html（永不覆盖 frontend 侧的 obsidian-bridge.js）
$s = Join-Path $SrcRoot "笔记主页.html"
$d = Join-Path $DstRoot "笔记主页.html"
if (Test-Path $d) {
    $hs = (Get-FileHash $s -Algorithm SHA256).Hash
    $hd = (Get-FileHash $d -Algorithm SHA256).Hash
    if ($hs -eq $hd) { $unchanged += "笔记主页.html" }
    elseif ($Verify) { Write-Host "[DIFF] 笔记主页.html" -ForegroundColor Yellow }
    else { Copy-Item $s $d -Force; $copied += "笔记主页.html" }
} else {
    if ($Verify) { Write-Host "[MISS] 笔记主页.html" -ForegroundColor Yellow }
    else { Copy-Item $s $d -Force; $copied += "笔记主页.html" }
}

# static/ vendor 目录递归同步（本地化库：markdown-it/hljs/codemirror）
$staticSrc = Join-Path $SrcRoot "static"
$staticDst = Join-Path $DstRoot "static"
if (Test-Path $staticSrc) {
    Get-ChildItem $staticSrc -Recurse -File | ForEach-Object {
        $rel = "static\" + $_.FullName.Substring($staticSrc.Length + 1)
        $s2 = $_.FullName
        $d2 = Join-Path $DstRoot $rel
        $dir = Split-Path $d2 -Parent
        if (-not (Test-Path $dir)) { New-Item -ItemType Directory -Force -Path $dir | Out-Null }
        if (Test-Path $d2) {
            $hs = (Get-FileHash $s2 -Algorithm SHA256).Hash
            $hd = (Get-FileHash $d2 -Algorithm SHA256).Hash
            if ($hs -eq $hd) { $unchanged += $rel; return }
            if ($Verify) { Write-Host "[DIFF] $rel" -ForegroundColor Yellow; return }
        } elseif ($Verify) {
            Write-Host "[MISS] $rel" -ForegroundColor Yellow
            return
        }
        Copy-Item $s2 $d2 -Force
        $copied += $rel
    }
}

Write-Host ""
if ($Verify) {
    Write-Host "校验模式：复制 $($copied.Count) / 未变 $($unchanged.Count) / 缺失 $($missing.Count)"
} else {
    Write-Host "同步完成：复制 $($copied.Count) 个文件"
    Write-Host "  已复制: $(if($copied){$copied -join '; '}else{'（无）'})"
}
Write-Host "  未变更(跳过): $($unchanged.Count) 个"