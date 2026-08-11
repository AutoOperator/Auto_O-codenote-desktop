package main

import (
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend
var frontendFS embed.FS

func main() {
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
