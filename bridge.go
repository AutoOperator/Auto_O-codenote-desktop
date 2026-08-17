package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
	"unsafe"
)

var (
	dwmapiDll                 = syscall.NewLazyDLL("dwmapi.dll")
	procDwmSetWindowAttribute = dwmapiDll.NewProc("DwmSetWindowAttribute")
)

// DWM 窗口属性（Windows 11 支持标题栏深浅/着色）
const (
	dwmwaUseImmersiveDarkMode = 20 // 标题栏深浅模式（BOOL；Win11 标准可靠 API）
	dwmwaBorderColor          = 34 // 窗口边框颜色
	dwmwaCaptionColor         = 35 // 标题栏背景颜色
	dwmwaTextColor            = 36 // 标题栏文字颜色
)

// Bridge 是 C 桥接（桌面版 Go 壳），实现与 obsidian-bridge.js 等价的 NR_OB 接口。
// A+C 架构：通用核心 A（frontend/笔记主页.html）通过 window.NR_OB 访问环境能力，
// 桌面版由本桥接提供——A 侧零改动即复用。方法经 wails.Bind 暴露为
// window.go.main.Bridge.*（wails 运行时绑定：包名小写 + 类型名）。
type Bridge struct {
	ctx      context.Context
	notesDir string  // 笔记根目录（默认 exe 同级 notes/）
	mainHwnd uintptr // 主窗口句柄（首次枚举缓存）
}

func NewBridge() *Bridge {
	exe, err := os.Executable()
	dir := ""
	if err == nil {
		dir = filepath.Join(filepath.Dir(exe), "notes")
	}
	return &Bridge{notesDir: dir}
}

func (b *Bridge) SetContext(ctx context.Context) { b.ctx = ctx }

// ensureNotesDir 确保笔记根目录存在（惰性创建），返回其路径
func (b *Bridge) ensureNotesDir() (string, error) {
	if b.notesDir == "" {
		return "", errors.New("无法定位笔记根目录（os.Executable 失败）")
	}
	if err := os.MkdirAll(b.notesDir, 0755); err != nil {
		return "", err
	}
	return b.notesDir, nil
}

// IsObsidian 桌面宿主环境恒真——A 侧 6 处调用点（保存/打开/设置/writeMode/录入）
// 以 isObsidian() 为 true 判定走 NR_OB 通道。
func (b *Bridge) IsObsidian() bool { return true }

// VaultPath 笔记根目录（exe 同级 notes/，自动创建）
func (b *Bridge) VaultPath() string {
	dir, err := b.ensureNotesDir()
	if err != nil {
		return ""
	}
	return dir
}

// GetSettings 返回 JSON {"vaultName": <目录名>, "vaultPath": <完整路径>}
func (b *Bridge) GetSettings() string {
	dir, err := b.ensureNotesDir()
	if err != nil {
		return `{"vaultName":"","vaultPath":""}`
	}
	out, _ := json.Marshal(map[string]string{
		"vaultName": filepath.Base(dir),
		"vaultPath": dir,
	})
	return string(out)
}

// SaveNote 写笔记文件。metaJSON={path,...}，path 为 vault 相对路径（自动建目录链）；
// 成功返回 {"path": 完整路径}。错误经 wails 绑定使 JS Promise reject（与 A 侧 try/catch 匹配）。
func (b *Bridge) SaveNote(md string, metaJSON string) (map[string]string, error) {
	var meta struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(metaJSON), &meta); err != nil {
		return nil, err
	}
	if meta.Path == "" {
		return nil, errors.New("saveNote: meta.path 为空")
	}
	full, err := b.writeRelative(meta.Path, md)
	if err != nil {
		return nil, err
	}
	return map[string]string{"path": full}, nil
}

// OpenNote 系统默认应用打开笔记文件。path 为 vault 相对路径（优先）；
// path 为空时按 id 在笔记目录搜索 <id>.md / <id>_*.md。文件不存在返回 false（A 侧降级）。
func (b *Bridge) OpenNote(id string, title string, path string) (bool, error) {
	dir, err := b.ensureNotesDir()
	if err != nil {
		return false, err
	}
	full := ""
	if path != "" {
		full = filepath.Join(dir, filepath.FromSlash(path))
	} else if id != "" {
		// 递归搜索 <id>.md / <id>_*.md（笔记在 代码笔记/算法笔记/<平台>/ 多级子目录）
		filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || full != "" {
				return nil
			}
			base := d.Name()
			if base == id+".md" || strings.HasPrefix(base, id+"_") && strings.HasSuffix(base, ".md") {
				full = p
			}
			return nil
		})
	} else {
		return false, errors.New("openNote: 缺少 id/path")
	}
	if full == "" {
		return false, nil
	}
	if _, err := os.Stat(full); err != nil {
		return false, nil // 文件不存在 → A 侧降级到其他打开方式
	}
	// 系统默认应用打开（Windows：cmd /c start "" "path"）
	cmd := exec.Command("cmd", "/c", "start", "", full)
	if err := cmd.Start(); err != nil {
		return false, err
	}
	return true, nil
}

// Fetch 网络抓取——C 提供网络能力：Wails 环境无 window.app.vault 且 UA 无 Electron，
// A 侧直抓判定（bridgeApp||isElectron）两者皆 false，抓取被迫走 CORS 代理
// （allorigins.win/Jina，国内网络不稳定）。由本方法代发 HTTP 请求（Go 无浏览器
// CORS 限制，直连目标站），返回 {"status": <状态码>, "body": <响应体>}。
// 响应体按 UTF-8 解码原样返回（HTML/JSON 均适用，非法序列替换为 U+FFFD）。
// 站点特化：洛谷 API 必需 x-lentille-request 头；Jina Reader 指定纯文本返回
// （与 A 侧原生抓取所带 headers 对齐）。
// CookieJar 必需：洛谷对无 cookie 的请求回 302 循环（先种 cookie 再放行），
// 无 Jar 时每次请求都是"首次"，永远 302——实测 200 需带 cookie。
func (b *Bridge) Fetch(rawURL string) (map[string]string, error) {
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Timeout: 15 * time.Second, Jar: jar}
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36")
	host := ""
	if u, err := url.Parse(rawURL); err == nil {
		host = u.Host
	}
	if strings.Contains(host, "luogu.com.cn") {
		req.Header.Set("x-lentille-request", "content-only")
	}
	if strings.Contains(host, "r.jina.ai") {
		req.Header.Set("Accept", "text/plain")
		req.Header.Set("X-Return-Format", "text")
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	body := string(raw)
	if !utf8.Valid(raw) {
		body = strings.ToValidUTF8(body, "�")
	}
	return map[string]string{
		"status": fmt.Sprintf("%d", resp.StatusCode),
		"body":   body,
	}, nil
}

// RecordQuestion 题目录入写文件。qJSON={path,md,...}，md 为 A 侧组装完成的完整笔记内容。
func (b *Bridge) RecordQuestion(qJSON string) (map[string]string, error) {
	var q struct {
		Path string `json:"path"`
		Md   string `json:"md"`
	}
	if err := json.Unmarshal([]byte(qJSON), &q); err != nil {
		return nil, err
	}
	if q.Path == "" {
		return nil, errors.New("recordQuestion: path 为空")
	}
	if q.Md == "" {
		return nil, errors.New("recordQuestion: md 为空")
	}
	full, err := b.writeRelative(q.Path, q.Md)
	if err != nil {
		return nil, err
	}
	return map[string]string{"path": full}, nil
}

// colorRef #RRGGBB → COLORREF（0x00bbggrr）
func colorRef(hex string) (uint32, error) {
	h := strings.TrimPrefix(strings.TrimSpace(hex), "#")
	if len(h) != 6 {
		return 0, errors.New("颜色格式错误: " + hex)
	}
	var r, g, b uint32
	if _, err := fmt.Sscanf(h, "%02x%02x%02x", &r, &g, &b); err != nil {
		return 0, err
	}
	return (b << 16) | (g << 8) | r, nil
}

// mainWindow 主窗口句柄：EnumWindows 找自身进程首个可见顶层窗口（首次枚举后缓存；
// wails v2 无公开窗口句柄 API——按 PID 枚举最可靠，与单实例唤起同方案）
func (b *Bridge) mainWindow() (uintptr, error) {
	if b.mainHwnd != 0 {
		return b.mainHwnd, nil
	}
	pid := uint32(os.Getpid())
	var found uintptr
	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		var wpid uint32
		procGetWindowThreadPid.Call(hwnd, uintptr(unsafe.Pointer(&wpid)))
		if wpid == pid {
			vis, _, _ := procIsWindowVisible.Call(hwnd)
			if vis != 0 {
				found = hwnd
				return 0
			}
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	if found == 0 {
		return 0, errors.New("未找到窗口")
	}
	b.mainHwnd = found
	return found, nil
}

// SetTitleBarMode 标题栏深浅模式跟随应用主题（DWM DWMWA_USE_IMMERSIVE_DARK_MODE，
// Win11 标准可靠 API：dark=true 深色标题栏浅色文字，false 浅色标题栏深色文字）。
// 实现对齐 wails v2 内部（win32.SetTheme）：int32 值（BOOL 语义）、属性 20
// （Win10 18985+/Win11 均支持；更早版本用 19 但 wails 最低支持 17763——本机 Win11 用 20）。
// 失败返回错误（A 侧静默忽略——浏览器/Obsidian 环境无此能力）。
func (b *Bridge) SetTitleBarMode(dark bool) error {
	hwnd, err := b.mainWindow()
	if err != nil {
		return err
	}
	var v int32 // wails 内部同款：int32（BOOL）
	if dark {
		v = 1
	}
	ret, _, _ := procDwmSetWindowAttribute.Call(hwnd, dwmwaUseImmersiveDarkMode, uintptr(unsafe.Pointer(&v)), unsafe.Sizeof(v))
	if int32(ret) != 0 {
		return fmt.Errorf("DwmSetWindowAttribute 失败: 0x%x", uint32(ret))
	}
	return nil
}

// SetTitleBarColor 旧版按背景深浅推断并转 SetTitleBarMode（自定义色 DWM 在部分
// Win11 版本不生效，深浅模式为标准可靠路径；保留兼容 A 侧旧调用）
func (b *Bridge) SetTitleBarColor(bgHex string, textHex string) error {
	bg, err := colorRef(bgHex)
	if err != nil {
		return err
	}
	// 背景亮度判定（BT.601 亮度权重）：暗背景 → 深色标题栏
	r := (bg >> 16) & 0xFF
	g := (bg >> 8) & 0xFF
	bl := bg & 0xFF
	luma := (299*r + 587*g + 114*bl) / 1000
	return b.SetTitleBarMode(luma < 128)
}

// OpenURL 系统默认浏览器打开链接（关注按钮等外链；WebView2 内 window.open 无多窗口支持）
func (b *Bridge) OpenURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("openUrl: url 为空")
	}
	cmd := exec.Command("cmd", "/c", "start", "", rawURL)
	if err := cmd.Start(); err != nil {
		return err
	}
	return nil
}

// DeleteNote 删除笔记文件。path 为 vault 相对路径（writeRelative 同款路径校验防穿越）；
// 文件不存在返回 false（不报错——目标状态已达成）；删除成功返回 true。
func (b *Bridge) DeleteNote(path string) (bool, error) {
	dir, err := b.ensureNotesDir()
	if err != nil {
		return false, err
	}
	clean := filepath.Clean(filepath.FromSlash(path))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return false, errors.New("非法路径: " + path)
	}
	full := filepath.Join(dir, clean)
	if _, err := os.Stat(full); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if err := os.Remove(full); err != nil {
		return false, err
	}
	return true, nil
}

// DeleteNoteVariants 按题号删除全部变体笔记文件。
// id 前缀匹配 notes 目录下 <id>_*.md（同题号多次生成/重命名会产生多文件，
// 单文件删除会残留旧变体）；同时兜底删除 path 指定的单文件。
// 返回 JSON 字符串 {"ok":bool,"removed":int}（单返回值，规避 wails 多返回值绑定歧义）：
//   - 文件不存在（幽灵项/已删过）→ ok=true（目标已达成），不误报失败
//   - 文件存在且删除成功 → ok=true, removed=n
//   - 文件存在但删除失败 → ok=false, removed=n
func (b *Bridge) DeleteNoteVariants(id string, path string) string {
	// 调试日志：记录调用参数（exe 同级 delete_debug.log）
	{
		exe, _ := os.Executable()
		logPath := filepath.Join(filepath.Dir(exe), "delete_debug.log")
		entry := time.Now().Format("2006-01-02 15:04:05.000") + " id=" + id + " path=" + path + "\n"
		os.WriteFile(logPath, []byte(entry), 0644)
	}
	dir, err := b.ensureNotesDir()
	if err != nil {
		return "{\"ok\":false,\"removed\":0}"
	}
	removed := 0
	failErr := error(nil)
	if id != "" {
		prefix := id + "_"
		filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			name := d.Name()
			if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".md") {
				if e := os.Remove(p); e != nil {
					if failErr == nil {
						failErr = e
					}
				} else {
					removed++
				}
			}
			return nil
		})
	}
	// 兜底：note_path 单文件（id 匹配未覆盖时）
	if removed == 0 && path != "" {
		clean := filepath.Clean(filepath.FromSlash(path))
		if !filepath.IsAbs(clean) && clean != ".." && !strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			full := filepath.Join(dir, clean)
			if _, err := os.Stat(full); err == nil {
				if e := os.Remove(full); e != nil {
					if failErr == nil {
						failErr = e
					}
				} else {
					removed++
				}
			}
		}
	}
	if failErr != nil {
		return fmt.Sprintf("{\"ok\":false,\"removed\":%d}", removed)
	}
	return fmt.Sprintf("{\"ok\":true,\"removed\":%d}", removed)
}

// writeRelative 写 vault 相对路径文件（自动建目录链）；防目录穿越
func (b *Bridge) writeRelative(rel string, content string) (string, error) {
	dir, err := b.ensureNotesDir()
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("非法路径: " + rel)
	}
	full := filepath.Join(dir, clean)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(full, []byte(content), 0644); err != nil {
		return "", err
	}
	return full, nil
}
