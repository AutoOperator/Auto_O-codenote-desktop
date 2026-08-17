# Auto_O算法笔记

Auto_O算法笔记是一款本地优先的算法刷题笔记管理工具，提供笔记记录、题目录入与复习管理功能。
笔记内容可离线查看；AI 辅助与在线抓题功能需联网并使用 API Key。支持洛谷、牛客等平台的题目录入与
内容抓取，提供看板、导图、关系图谱等多种视图，内置纸感、玻璃两套主题（各含亮色与暗色）。

## 许可证

CC BY-NC 4.0（知识共享 署名-非商业性使用 4.0 国际）：允许学习、研究、个人使用、非商业分享与修改；
禁止任何形式的商业用途；使用或分发时须保留原作者署名。完整条款见 [LICENSE](./LICENSE) 与
https://creativecommons.org/licenses/by-nc/4.0/

## 功能

- **本地笔记**：代码笔记、算法笔记、题目模板，Markdown 存储于本机
- **题目录入**：洛谷 / 牛客等多平台在线抓题，自动生成规范 frontmatter
- **复习管理**：三轮复习机制 + 掌握度标记，题目分类管理
- **多视图**：列表、看板、导图、关系图谱，先修依赖可视化
- **主题**：玻璃 / 纸感两套，各含亮色与暗色
- **离线查看**：笔记内容本地可读；AI 辅助与在线抓题需联网并配置 API Key

## 界面一览

**主界面与主题**：顶部工具栏（新建 / 刷新 / 设置）、左侧知识点导航、中间笔记列表（圆点标记完成状态）、
右侧按知识点 / 难度 / 状态 / 平台多维统计筛选。

| 主题 | 亮色 | 暗色 |
|---|---|---|
| 玻璃 | ![玻璃 · 亮](assets/screenshots/1主界面（主题：玻璃 · 亮）.png) | ![玻璃 · 暗](assets/screenshots/2主界面（主题：玻璃 · 暗）.png) |
| 纸感 | ![纸感 · 亮](assets/screenshots/3主界面（主题：纸感 · 亮）.png) | ![纸感 · 暗](assets/screenshots/4主界面（主题：纸感 · 暗）.png) |

**笔记阅读**：左右分栏——左侧题面（描述 / 输入输出 / 样例），右侧代码区（语法高亮、保存、复制）。

![算法笔记阅读界面](assets/screenshots/5算法笔记阅读界面.png)

**题目录入**：支持链接抓题、图片转化、文本填写三种来源；可附自己的 AC 代码与官方 AC 代码，
一键生成规范 frontmatter 笔记。

![题目录入界面](assets/screenshots/6题目录入界面.png)

**看板视图**：按知识点 / 难度 / 状态分列组织卡片，标注平台与未完成数；支持拖拽跨列（高亮边框反馈），
点击卡片查看笔记。

![看板视图](assets/screenshots/7看板视图.png)　![看板拖拽](assets/screenshots/8看板视图（题目在看板间拖动）.png)

**列表视图**：按一级分类分组，支持搜索过滤、悬停查看简述，行内展示平台标识、题目编号与标题。

![列表视图](assets/screenshots/9列表视图.png)

**导图视图**：以「算法体系」为根展开知识层级，支持滚轮缩放、切换纵向布局、按难度 / 分类 / 题量着色筛选；
点击节点展开细分知识点，右侧列出关联题目（名称、别名、简述）。

![导图视图](assets/screenshots/10导图视图.png)　![导图展开](assets/screenshots/11导图视图（展开）.png)

![节点跳转](assets/screenshots/12导图视图（节点跳转相关题目）.png)

**关系图谱**：径向布局呈现知识点网络，虚线表示先修依赖；可一键隐藏虚线，支持难度 / 分类 / 题量维度。

![先修依赖](assets/screenshots/13关系图谱视图（展示先修依赖虚线）.png)　![隐藏依赖](assets/screenshots/15关系图谱视图（隐藏先修依赖）.png)

**左侧栏导航**：搜索 + 知识点树形导航（基础算法、搜索、动态规划、图论等），分类标注条目数；
点击条目可跳转 OI Wiki 对应页面。

![左侧栏](assets/screenshots/左侧栏.png)　![跳转OI Wiki](assets/screenshots/左侧栏跳转oiwiki界面.png)

## 下载与运行

从 [Releases](https://github.com/AutoOperator/Auto_O-codenote-desktop/releases) 下载最新版压缩包，
解压后运行 `Auto_O-codenote-desktop.exe` 即可（需 Windows 10 / 11，系统自带 WebView2 运行时）。
笔记数据存放于 exe 同级 `notes/` 目录，备份时复制该目录即可。

---

### 开发者信息

基于 Wails v2（Go + WebView2）构建，与 Obsidian 插件版共享前端核心，桌面端通过 Go 桥接层实现
本地文件读写与网络抓取。构建：`wails build`（产物 `build/bin/Auto_O-codenote-desktop.exe`）。

© 2026 Auto_Operator
