package main

import (
	"context"
	"embed"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var frontendFS embed.FS

func main() {
	// 禁用 WebView2 系统代理（避免 CDN 资源加载失败）+ 启用 CDP 调试端口（本地验收/调试用）
	os.Setenv("WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS", "--no-proxy-server --remote-debugging-port=9223")
	bridge := NewBridge()

	err := wails.Run(&options.App{
		Title:  "代码笔记",
		Width:  1200,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: frontendFS,
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
