package routes

import (
	"fmt"
	"ibsTool/htop"
	"ibsTool/logging"
	"ibsTool/models"
	"ibsTool/sniffer"
	systeminfo "ibsTool/systemInfo"
	"ibsTool/tailscale"
	"ibsTool/utils"
	"log"
	"net/http"
	"strings"
	"text/template"

	"github.com/zuadi/webServer"
	wsModels "github.com/zuadi/webServer/models"
)

func SetRoutes(s *webServer.WebServer, l *logging.Logger) error {

	ws := s.NewWebSocket("/ws")

	wsHtop := s.NewWebSocket("/wshtop")

	_, err := sniffer.NewModbusRTUSniffer(ws, l)
	if err != nil {
		return err
	}

	htopTask := htop.NewHtopProcess(wsHtop, l)

	// TODO: not yet implemented
	//	go sniffer.StartTCP(ws, l)

	s.ServeFile("/tailwind.js", "./html/tailwind.cdn.js")

	s.Get("/", func(ctx wsModels.Context) {
		w := ctx.GetResponseWriter()
		var tmpl *template.Template
		var err error

		tmpl, err = template.ParseFiles("./html/landingPage.html")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	})

	s.Get("/info", func(ctx wsModels.Context) {
		ctx.RespondJson(http.StatusOK, models.GetInfo())
	})

	s.Get("/system/info", func(ctx wsModels.Context) {
		sI := systeminfo.GetInfo()
		ctx.RespondJson(http.StatusOK, sI)
	})

	modbus := s.NewGroup("/modbus")
	rtuPort := modbus.NewGroup("/rtu")
	tcpPort := modbus.NewGroup("/tcp")

	rtuPort.Get("/", func(ctx wsModels.Context) {
		w := ctx.GetResponseWriter()
		var tmpl *template.Template
		var err error

		tmpl, err = template.ParseFiles("./html/modbusRTU.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	})

	rtuPort.Get("/ports", func(ctx wsModels.Context) {

		response := []models.Port{}

		ports, err := utils.GetSerialPorts()
		if err != nil {
			ctx.RespondJson(http.StatusBadRequest, fmt.Sprintf(`{"error":"%s"}`, err.Error()))
		}

		for _, p := range ports {
			label := strings.TrimSpace(fmt.Sprintf("%s %s", p.Product, p.Manufacturer))

			if label == "" {
				if strings.HasPrefix(p.Name, "/dev/ttyUSB") || strings.HasPrefix(p.Name, "/dev/ttyACM") {
					label = fmt.Sprintf("USB Serial (%s)", p.Name)
				} else if strings.HasPrefix(p.Name, "/dev/ttyS") {
					label = fmt.Sprintf("Hardware UART (%s)", p.Name)
				} else {
					label = p.Name
				}
			} else {
				label = fmt.Sprintf("%s (%s)", label, p.Name)
			}

			response = append(response, models.Port{
				Port:  p.Name,
				Label: label,
			})
		}
		ctx.RespondJson(http.StatusOK, response)
	})

	tcpPort.Get("/", func(ctx wsModels.Context) {
		w := ctx.GetResponseWriter()
		var tmpl *template.Template
		var err error

		tmpl, err = template.ParseFiles("./html/tcp.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	})

	htopGroup := s.NewGroup("/htop")
	htopGroup.Get("/", func(ctx wsModels.Context) {
		go htopTask.Start()
		w := ctx.GetResponseWriter()
		tmpl, err := template.ParseFiles("./html/htop.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	})

	htopGroup.Post("/settings", func(ctx wsModels.Context) {
		r := ctx.GetRequest()

		interval, err := models.GetInterval(r.Body)
		if err != nil {
			http.Error(ctx.GetResponseWriter(), err.Error(), http.StatusInternalServerError)
			return
		}
		htopTask.ChangeInterval(interval)
	})

	tailscaleGroup := s.NewGroup("/tailscale")

	tailscaleGroup.Get("/", func(ctx wsModels.Context) {
		w := ctx.GetResponseWriter()

		tmpl, err := template.ParseFiles("./html/tailscale.html")

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tmpl.Execute(w, nil)
	})

	ts, err := tailscale.NewTailscaleClient(l)
	if err != nil {
		log.Fatal(err)
	}

	tailscaleGroup.Post("/settings", func(ctx wsModels.Context) {
		data := &tailscale.Settings{}
		if err := models.ReadJsonBody(ctx, data); err != nil {
			ctx.RespondJson(http.StatusBadRequest, err.Error())
			return
		}

		switch data.Client {
		case "start":
			go ts.Connect()
		case "stop":
			ts.Disconnect()
		}
	})

	logs := s.NewGroup("/logs")

	// 1. Render Dashboard HTML Page
	logs.Get("/", func(ctx wsModels.Context) {
		w := ctx.GetResponseWriter()
		tmpl, err := template.ParseFiles("./html/logs.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = tmpl.Execute(w, nil)
	})

	// 2. Realtime SSE Log Stream Endpoint
	logs.Get("/stream", func(ctx wsModels.Context) {
		l.ConnectSSE(ctx)
	})

	// 3. Update MaxLogs Setting & Broadcast via SSE
	logs.Post("/settings", func(ctx wsModels.Context) {
		l.BroadcastConfig(ctx)
	})

	// used files
	s.ServeFile("/nav-drawer.js", "./html/nav-drawer.js")
	s.ServeFile("/tailwind.js", "./html/tailwind.cdn.js")

	l.BroadcastLog("🚀 Industrial Monitor Listening live")

	if err := s.ListenHttp(); err != nil {
		log.Fatal(err)
	}
	return nil
}
