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
	"time"
	"unicode/utf8"
)

// Bridge 是 C 桥接（桌面版 Go 壳），实现与 obsidian-bridge.js 等价的 NR_OB 接口。
// A+C 架构：通用核心 A（frontend/笔记主页.html）通过 window.NR_OB 访问环境能力，
// 桌面版由本桥接提供——A 侧零改动即复用。方法经 wails.Bind 暴露为
// window.go.main.Bridge.*（wails 运行时绑定：包名小写 + 类型名）。
type Bridge struct {
	ctx      context.Context
	notesDir string // 笔记根目录（默认 exe 同级 notes/）
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
