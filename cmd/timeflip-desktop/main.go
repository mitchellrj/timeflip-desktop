package main

import (
	"context"
	"log"
	"os"

	"github.com/mitchellrj/timeflip-desktop/internal/app"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	controller, bus, err := app.BuildController(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	err = wails.Run(&options.App{
		Title:  "TimeFlip Desktop",
		Width:  1100,
		Height: 760,
		AssetServer: &assetserver.Options{
			Assets: os.DirFS("frontend/dist"),
		},
		BackgroundColour: &options.RGBA{R: 245, G: 247, B: 250, A: 1},
		OnStartup: func(ctx context.Context) {
			bus.SetContext(ctx)
			controller.Startup(ctx)
		},
		OnShutdown: controller.Shutdown,
		Bind: []any{
			controller,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
