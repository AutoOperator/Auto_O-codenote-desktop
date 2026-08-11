package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

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
	// 禁用 WebView2 系统代理（避免 CDN 资源加载失败）+ 启用 CDP 调试端口（本地验收/调试用）
	os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "--no-proxy-server --remote-debugging-port=9223")
	bridge := NewBridge()

	err := wails.Run(&options.App{
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
