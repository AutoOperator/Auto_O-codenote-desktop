# code-note-desktop — 代码笔记桌面版

代码笔记（算法笔记）的桌面版应用：本地 Markdown 笔记 + 题目录入 + 复习工具。
基于 Wails v2（Go + WebView2）构建，Windows 原生桌面体验，无服务器、数据全本地。

## 📜 许可证（重要）

**CC BY-NC 4.0（知识共享 署名-非商业性使用 4.0 国际）**

- ✅ 允许：学习、研究、个人使用、非商业分享与修改
- ❌ 禁止：任何形式的商业用途（售卖、商业集成、盈利性部署）
- 📌 要求：使用或分发时必须保留原作者署名 **Auto_Operator**

完整文本见 [LICENSE](./LICENSE) 与 https://creativecommons.org/licenses/by-nc/4.0/

## ✨ 功能

- 本地 Markdown 笔记管理（代码笔记 / 算法笔记 / 题目模板）
- 题目录入（洛谷 / 牛客等多平台，支持网络抓题）
- CodeMirror 代码高亮 + Markdown 渲染 + 手写笔记板
- 多种主题（paper-dark 等）+ 深浅色标题栏自适应
- 笔记数据存于 exe 同级 `notes/` 目录，完全离线

## 🏗 架构（A+C）

- **A（通用核心）**：`frontend/笔记主页.html` + `frontend/static/` —— 单文件应用核心（随仓库发布）
- **C（Go 桥接）**：`bridge.go` —— 通过 wails.Bind 暴露 `window.go.main.Bridge.*`，
  实现与 Obsidian 版等价的 `window.NR_OB` 接口（isObsidian/vaultPath/saveNote/openNote/recordQuestion/getSettings/fetch）

## 📁 目录结构

```
code-note-desktop/
├── main.go            # Wails v2 入口（//go:embed all:frontend + AssetServer）
├── bridge.go          # C 桥接（NR_OB 等价接口，wails.Bind 暴露）
├── go.mod / wails.json
├── sync_frontend.ps1  # 从 code-note-obsidian 幂等同步 A（单一来源，仓库内为发布快照）
├── LICENSE            # CC BY-NC 4.0
├── assets/            # 应用图标源
│   ├── appicon.png    # 512x512 定稿图标
│   └── appicon.svg    # 矢量源
├── build/windows/     # wails 构建模板（icon.ico / info.json / manifest）
└── frontend/
    ├── 笔记主页.html        # A 侧核心（随仓库发布）
    ├── obsidian-bridge.js   # C 包装（window.go.* → window.NR_OB）
    └── static/              # 离线 vendor（markdown-it/hljs/codemirror）
```

> 图标：改图标后执行 `powershell -File assets/make-icon.ps1`
> （生成 build/appicon.png + build/windows/icon.ico 多尺寸），再 `wails build`。

## 🛠 开发 / 构建

前置：Go 1.26+、Wails v2 CLI（`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）、WebView2 Runtime（Win10/11 自带）

```powershell
# 开发运行
wails dev

# 构建（产物 build/bin/code-note-desktop.exe）
wails build
```

笔记数据落在 exe 同级 `notes/` 目录（首次运行自动创建），
路径约定：`代码笔记/算法笔记/<平台>/<id>_<标题>.md`。

## 🔌 桥接接口（window.NR_OB）

| 方法 | 说明 |
|---|---|
| isObsidian() | 桌面宿主恒真（A 侧据此走 NR_OB 通道） |
| vaultPath() | 笔记根目录（exe 同级 notes/） |
| saveNote(md, meta) | 写笔记文件（meta.path 相对路径，自动建目录链） |
| openNote(id, title, path) | 系统默认应用打开（无 path 按 id 搜索） |
| recordQuestion(q) | 题目录入写文件（q={path,md}） |
| getSettings() | JSON {vaultName, vaultPath} |
| fetch(url) | 网络抓取（Go 无 CORS 直连，返回 fetch Response 兼容子集） |
| setTitleBarMode(dark) | 标题栏深浅模式（Win11 DWM） |
| deleteNote(path) | 安全删除笔记（防目录穿越） |

---
© 2026 Auto_Operator · [CC BY-NC 4.0](./LICENSE)
