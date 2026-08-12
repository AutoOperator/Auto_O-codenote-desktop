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
	"strings"
	"syscall"
	"unsafe"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

var (
	kernel32Dll       = syscall.NewLazyDLL("kernel32.dll")
	user32Dll         = syscall.NewLazyDLL("user32.dll")
	procCreateMutexW  = kernel32Dll.NewProc("CreateMutexW")
	procFindWindowW   = user32Dll.NewProc("FindWindowW")
	procShowWindow    = user32Dll.NewProc("ShowWindow")
	procSetForeground = user32Dll.NewProc("SetForegroundWindow")
)

const swRestore = 9 // SW_RESTORE：还原最小化/最大化窗口

// ensureSingleInstance 单实例保护：命名互斥量（名称含 exe 路径 hash——源码版/测试版
// 不同路径互不干扰）。已存在实例时唤起其主窗口（FindWindowW 按标题「代码笔记」）后退出；
// 本实例持有互斥量句柄直至进程结束（返回的释放函数由 main defer 调用）。
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
		title, _ := syscall.UTF16PtrFromString("代码笔记")
		hwnd, _, _ := procFindWindowW.Call(0, uintptr(unsafe.Pointer(title)))
		if hwnd != 0 {
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

	err = wails.Run(&options.App{
		Title:  "代码笔记",
		Width:  1200,
		Height: 800,
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
