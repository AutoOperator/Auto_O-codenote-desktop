# code-note-desktop — 代码笔记桌面版（A+C 架构）

通用核心 A（`code-note-obsidian/笔记主页.html`）的桌面 Wails 壳。
C（Go 桥接）实现与 Obsidian 版（B 桥接）等价的 NR_OB 接口——A 侧零改动即复用。

## 架构

```
A（通用核心）   code-note-obsidian/笔记主页.html + static/   ← 单一来源，sync 复制
C（Go 桥接）    bridge.go（wails.Bind → window.go.main.Bridge.*）
前端包装        frontend/obsidian-bridge.js（window.go.* → window.NR_OB）
```

A 侧通过 `window.NR_OB` 访问环境能力（isObsidian/vaultPath/saveNote/openNote/recordQuestion/getSettings）。
Obsidian 环境由 `obsidian-bridge.js`（B 桥接）提供；桌面版由本壳（C 桥接）提供等价接口。

## 目录结构

```
code-note-desktop/
├── main.go            # Wails v2 入口（//go:embed all:frontend + AssetServer）
├── bridge.go          # C 桥接（NR_OB 等价接口，wails.Bind 暴露）
├── go.mod / wails.json
├── sync_frontend.ps1  # 从 code-note-obsidian 幂等复制 A（笔记主页.html + static/）
└── frontend/
    ├── 笔记主页.html        # A（sync 复制，不入 git）
    ├── obsidian-bridge.js   # C 包装（本仓库维护）
    └── static/              # 离线 vendor（sync 复制，不入 git）
```

## 开发

```powershell
# 1) 同步前端 A（源变更后必跑；SHA256 幂等）
powershell -File sync_frontend.ps1            # 执行同步
powershell -File sync_frontend.ps1 -Verify    # 只比对报告

# 2) 开发运行（wails CLI）
wails dev

# 3) 构建
wails build
```

笔记数据落在 exe 同级 `notes/` 目录（首次运行自动创建），
路径约定与 Obsidian 版一致：`代码笔记/算法笔记/<平台>/<id>_<标题>.md`。

## 桥接接口（window.NR_OB）

| 方法 | 说明 |
|---|---|
| isObsidian() | 桌面宿主恒真（A 侧据此走 NR_OB 通道） |
| vaultPath() | 笔记根目录（exe 同级 notes/） |
| saveNote(md, meta) | 写笔记文件（meta.path 相对路径，自动建目录链） |
| openNote(id, title, path) | 系统默认应用打开（无 path 按 id 搜索） |
| recordQuestion(q) | 题目录入写文件（q={path,md}） |
| getSettings() | JSON {vaultName, vaultPath} |
