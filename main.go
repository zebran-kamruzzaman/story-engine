package main

import (
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	storyApp := &App{}

	app := application.New(application.Options{
		Name:        "Story Engine",
		Description: "A quiet narrative writing assistant",
		Services: []application.Service{
			application.NewService(storyApp),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	win := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Story Engine",
		Width:            1440,
		Height:           900,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: application.NewRGBA(27, 27, 27, 255),
		URL:              "/",
	})

	// Initialize all services BEFORE app.Run().
	// This guarantees services are ready before the WebView loads and the
	// frontend fires its first IPC call — no race condition possible.
	if err := storyApp.Initialize(win); err != nil {
		log.Fatalf("failed to initialize: %v", err)
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
