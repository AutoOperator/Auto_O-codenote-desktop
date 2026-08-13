package main

import (
	"context"
	"crypto/md5"
	"embed"
	"encoding/hex"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	wailsWindows "github.com/wailsapp/wails/v2/pkg/options/windows"
)

var (
	kernel32Dll              = syscall.NewLazyDLL("kernel32.dll")
	user32Dll                = syscall.NewLazyDLL("user32.dll")
	procCreateMutexW         = kernel32Dll.NewProc("CreateMutexW")
	procEnumWindows          = user32Dll.NewProc("EnumWindows")
	procGetWindowThreadPid   = user32Dll.NewProc("GetWindowThreadProcessId")
	procIsWindowVisible      = user32Dll.NewProc("IsWindowVisible")
	procShowWindow           = user32Dll.NewProc("ShowWindow")
	procSetForeground        = user32Dll.NewProc("SetForegroundWindow")
	procOpenProcess          = kernel32Dll.NewProc("OpenProcess")
	procQueryFullProcessName = kernel32Dll.NewProc("QueryFullProcessImageNameW")
)

const (
	swRestore               = 9      // SW_RESTORE：还原最小化/最大化窗口
	processQueryLimitedInfo = 0x1000 // PROCESS_QUERY_LIMITED_INFORMATION
)

// processPath 查询进程 exe 完整路径（QueryFullProcessImageNameW）
func processPath(pid uintptr) string {
	h, _, _ := procOpenProcess.Call(processQueryLimitedInfo, 0, pid)
	if h == 0 {
		return ""
	}
	defer syscall.CloseHandle(syscall.Handle(h))
	var buf [512]uint16
	size := uint32(len(buf))
	ok, _, _ := procQueryFullProcessName.Call(h, 0, uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&size)))
	if ok == 0 {
		return ""
	}
	return syscall.UTF16ToString(buf[:size])
}

// findExistingWindow 枚举顶层窗口，找与当前 exe 同路径进程的可见窗口
// （wails 窗口标题含编码问题，按标题 FindWindowW 不可靠——按 exe 路径匹配最稳；
//
//	互斥量名已按 exe 路径 hash 隔离版本，同互斥量即同路径）
func findExistingWindow(exePath string) uintptr {
	var found uintptr
	cb := syscall.NewCallback(func(hwnd uintptr, lparam uintptr) uintptr {
		var pid uint32
		procGetWindowThreadPid.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
		if pid == 0 {
			return 1
		}
		if strings.EqualFold(filepath.Clean(processPath(uintptr(pid))), filepath.Clean(exePath)) {
			// 优先可见顶层窗口（主窗口），隐藏工具窗口跳过
			vis, _, _ := procIsWindowVisible.Call(hwnd)
			if vis != 0 {
				found = hwnd
				return 0 // 停止枚举
			}
			if found == 0 {
				found = hwnd // 兜底记录首个同路径窗口
			}
		}
		return 1
	})
	procEnumWindows.Call(cb, 0)
	return found
}

// ensureSingleInstance 单实例保护：命名互斥量（名称含 exe 路径 hash——源码版/测试版
// 不同路径互不干扰）。已存在实例时唤起其主窗口（EnumWindows 按 exe 路径匹配，
// 还原 + 置前台）后退出；本实例持有互斥量句柄直至进程结束（返回的释放函数由 main defer 调用）。
func ensureSingleInstance() (func(), error) {
	exePath := ""
	if exe, err := os.Executable(); err == nil {
		exePath = exe
	}
	sum := md5.Sum([]byte(exePath))
	name := "Local\\CodeNoteDesktop_" + hex.EncodeToString(sum[:8])
	mutexName, _ := syscall.UTF16PtrFromString(name)
	h, _, errCode := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(mutexName)))
	if h == 0 {
		return nil, fmt.Errorf("创建互斥量失败: %v", errCode)
	}
	if errCode == syscall.ERROR_ALREADY_EXISTS {
		// 已有实例：唤起其主窗口（还原 + 置前台），本进程直接退出
		if hwnd := findExistingWindow(exePath); hwnd != 0 {
			procShowWindow.Call(hwnd, swRestore)
			procSetForeground.Call(hwnd)
		}
		syscall.CloseHandle(syscall.Handle(h))
		os.Exit(0)
	}
	return func() { syscall.CloseHandle(syscall.Handle(h)) }, nil
}

//go:embed all:frontend
var frontendFS embed.FS

// assetHandler 服务 frontend 嵌入文件系统；根路径返回默认入口 笔记主页.html
// （wails 无 IndexHTML 配置项，A 入口为中文文件名，需自定义 Handler 指定）
func assetHandler() http.Handler {
	sub, _ := fs.Sub(frontendFS, "frontend")
	fileServer := http.FileServerFS(sub)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" || p == "/" {
			r.URL.Path = "/笔记主页.html"
		}
		fileServer.ServeHTTP(w, r)
	})
}

func main() {
	// 单实例：已存在实例时唤起其窗口并退出；句柄保持至进程结束
	releaseMutex, err := ensureSingleInstance()
	if err != nil {
		log.Fatalf("单实例检查失败: %v", err)
	}
	defer releaseMutex()
	// 禁用 WebView2 系统代理（避免 CDN 资源加载失败）+ 启用 CDP 调试端口（本地验收/调试用）
	os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "--no-proxy-server --remote-debugging-port=9223")
	bridge := NewBridge()

	// 标题栏主题兜底（启动静态配置，对齐 wails ThemeSettings；动态切换由 A 侧
	// DWM 调用实现）：默认主题 paper-dark → 深色标题栏，浅色主题对应浅色配置
	titleTheme := &wailsWindows.ThemeSettings{
		DarkModeTitleBar:           wailsWindows.RGB(0x1a, 0x1d, 0x23),
		DarkModeTitleBarInactive:   wailsWindows.RGB(0x1a, 0x1d, 0x23),
		DarkModeTitleText:          wailsWindows.RGB(0xd8, 0xdd, 0xe4),
		DarkModeTitleTextInactive:  wailsWindows.RGB(0x9a, 0xa3, 0xb0),
		DarkModeBorder:             wailsWindows.RGB(0x2a, 0x2f, 0x38),
		DarkModeBorderInactive:     wailsWindows.RGB(0x2a, 0x2f, 0x38),
		LightModeTitleBar:          wailsWindows.RGB(0xed, 0xe4, 0xd3),
		LightModeTitleBarInactive:  wailsWindows.RGB(0xed, 0xe4, 0xd3),
		LightModeTitleText:         wailsWindows.RGB(0x3a, 0x35, 0x30),
		LightModeTitleTextInactive: wailsWindows.RGB(0x5c, 0x52, 0x48),
		LightModeBorder:            wailsWindows.RGB(0xcd, 0xbf, 0xa8),
		LightModeBorderInactive:    wailsWindows.RGB(0xcd, 0xbf, 0xa8),
	}
	err = wails.Run(&options.App{
		Title:  "Auto_O算法笔记",
		Width:  1200,
		Height: 800,
		Windows: &wailsWindows.Options{
			Theme:       wailsWindows.Dark, // 启动默认深色标题栏（与 A 侧默认主题 paper-dark 一致）
			CustomTheme: titleTheme,
		},
		AssetServer: &assetserver.Options{
			Handler: assetHandler(),
		},
		OnStartup: func(ctx context.Context) {
			bridge.SetContext(ctx)
		},
		Bind: []interface{}{
			bridge,
		},
	})
	if err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}
