package main

import (
	"log"

	"ibsTool/logging"
	"ibsTool/routes"

	"github.com/joho/godotenv"

	"github.com/zuadi/webServer"
)

// --- MAIN RUNTIME SERVER ---
func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	s := webServer.NewWebServer("0.0.0.0", 8000)
	lgr := logging.NewLogger()

	lgr.BroadcastLog("start IBS-Tool")

	err = routes.SetRoutes(s, lgr)
	if err != nil {
		log.Fatal(err.Error())
	}

	lgr.BroadcastLog("🚀 Industrial Monitor Listening live at http://localhost:8000")

	if err := s.ListenHttp(); err != nil {
		log.Fatal(err)
	}
}
