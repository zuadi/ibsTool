//go:build buildtool

package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"ibsTool/logging"
	"ibsTool/routes"
	"ibsTool/udpDiscovery"

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

	// start discover udp port
	go func() {
		fmt.Println(100)
		if err := udpDiscovery.UPDListener(lgr); err != nil {
			fmt.Println(66)
			lgr.BroadcastLog(err)
		}
		fmt.Println(200)

	}()

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
