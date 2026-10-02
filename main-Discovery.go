//go:build discovery

package main

import (
	"embed"
	udpdiscovery "ibsTool/udpDiscovery"
	"log"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	app := udpdiscovery.NewApp()

	err := wails.Run(&options.App{
		Title:  "IBS-Tool Finder",
		Width:  800,
		Height: 600,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: app.Startup,
		Bind: []interface{}{
			app,
		},
	})

	if err != nil {
		log.Fatal("Error:", err.Error())
	}
}
