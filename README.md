# Auto_O算法笔记

一款本地优先的算法刷题笔记工具：记笔记、录题目、管复习，数据全在你自己的电脑里。

支持洛谷 / 牛客等平台的题目录入与抓取，自带看板、导图、关系图谱等多种视图，
配合纸感 / 玻璃主题，让刷题记录清晰又顺手。

## 界面一览

四种主题，按你的喜好切换：

| 主题 | 亮色 | 暗色 |
|---|---|---|
| 玻璃 | ![玻璃亮](assets/screenshots/1主界面（主题：玻璃 · 亮）.png) | ![玻璃暗](assets/screenshots/2主界面（主题：玻璃 · 暗）.png) |
| 纸感 | ![纸感亮](assets/screenshots/3主界面（主题：纸感 · 亮）.png) | ![纸感暗](assets/screenshots/4主界面（主题：纸感 · 暗）.png) |

记录与复习：

![算法笔记阅读界面](assets/screenshots/5算法笔记阅读界面.png)

![题目录入界面](assets/screenshots/6题目录入界面.png)

多种视图，怎么顺眼怎么来：

![看板视图](assets/screenshots/7看板视图.png)　![看板拖拽](assets/screenshots/8看板视图（题目在看板间拖动）.png)

![列表视图](assets/screenshots/9列表视图.png)　![导图视图](assets/screenshots/10导图视图.png)

![导图展开](assets/screenshots/11导图视图（展开）.png)　![节点跳转](assets/screenshots/12导图视图（节点跳转相关题目）.png)

![先修依赖](assets/screenshots/13关系图谱视图（展示先修依赖虚线）.png)　![隐藏依赖](assets/screenshots/15关系图谱视图（隐藏先修依赖）.png)

![左侧栏](assets/screenshots/左侧栏.png)　![跳转OI Wiki](assets/screenshots/左侧栏跳转oiwiki界面.png)

## 功能

- 本地 Markdown 笔记：代码笔记、算法笔记、题目模板，数据存在自己电脑
- 题目录入：洛谷 / 牛客等多平台，支持在线抓题，自动生成规范 frontmatter
- 复习流程：三轮复习 + 掌握度标记，题目分类管理
- 多视图：列表 / 看板 / 导图 / 关系图谱，先修依赖一目了然
- 主题切换：玻璃 / 纸感，亮 / 暗自适应
- 完全离线可用：无需登录、无云端依赖

## 下载与运行

从 [Releases](https://github.com/AutoOperator/Auto_O-codenote-desktop/releases) 下载最新版压缩包，
解压后双击 `Auto_O-codenote-desktop.exe` 即可使用（Windows 10 / 11，自带 WebView2）。

笔记数据保存在 exe 同级的 `notes/` 目录，备份时复制它即可。

## 许可证

CC BY-NC 4.0（知识共享 署名-非商业性使用 4.0 国际）：

- 允许：学习、研究、个人使用、非商业分享与修改
- 禁止：任何形式的商业用途
- 要求：使用或分发时保留原作者署名

完整条款见 [LICENSE](./LICENSE) 与 https://creativecommons.org/licenses/by-nc/4.0/

---

### 开发者信息

Auto_O算法笔记 桌面版基于 Wails v2（Go + WebView2）构建，与 Obsidian 插件版共享同一套
前端核心（A 侧），桌面壳通过 Go 桥接（C 侧）提供本地文件读写与网络抓取能力。

构建：`wails build`（产物 `build/bin/Auto_O-codenote-desktop.exe`），需 Go 1.26+ 与 Wails CLI。

© 2026 Auto_Operator
