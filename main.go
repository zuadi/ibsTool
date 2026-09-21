package main

import (
	"log"
	"os"
	"strconv"

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

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		log.Fatal(err)
	}

	s := webServer.NewWebServer("0.0.0.0", portInt)
	lgr := logging.NewLogger()

	lgr.BroadcastLog("start IBS-Tool")

	err = routes.SetRoutes(s, lgr)
	if err != nil {
		log.Fatal(err.Error())
	}

	lgr.BroadcastLog("🚀 Industrial Monitor Listening live at http://localhost:" + port)

	if err := s.ListenHttp(); err != nil {
		log.Fatal(err)
	}
}
