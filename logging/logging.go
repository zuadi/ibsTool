package logging

import (
	"fmt"
	"ibsTool/models"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	wsModels "github.com/zuadi/webServer/models"
)

type Logger struct {
	logHistory []string
	clients    map[chan models.SSEEvent]bool
	stateMutex sync.Mutex
	MaxLogs    int
}

func NewLogger() *Logger {
	var maxBuffer int = 500

	mB := os.Getenv("LOG_MAXBUFFER")
	if mB != "" {
		if v, err := strconv.Atoi(mB); err == nil {
			maxBuffer = v
		}
	}

	return &Logger{
		logHistory: make([]string, 0, maxBuffer),
		clients:    make(map[chan models.SSEEvent]bool),
		MaxLogs:    maxBuffer,
	}
}

func (l *Logger) ConnectSSE(ctx wsModels.Context) {
	w := ctx.GetResponseWriter()
	req := ctx.GetRequest()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Subscribe client to log broadcaster
	clientChan := l.subscribe()
	defer l.unsubscribe(clientChan)

	// Send existing log history immediately upon connect
	history := l.getHistory()
	if history != "" {
		for line := range strings.SplitSeq(history, "\n") {
			fmt.Fprintf(w, "data: %s\n\n", line)
		}
		flusher.Flush()
	}

	// Stream live events until client disconnects
	for {
		select {
		case <-req.Context().Done():
			return
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			b, err := msg.Bytes()
			if err != nil {
				log.Fatal(err)
			}
			_, _ = w.Write(b)
			flusher.Flush()
		}
	}
}

// Subscribe adds a client connection channel to the SSE stream
func (l *Logger) subscribe() chan models.SSEEvent {
	l.stateMutex.Lock()
	defer l.stateMutex.Unlock()

	ch := make(chan models.SSEEvent, 100)
	l.clients[ch] = true
	return ch
}

// Unsubscribe removes and closes a client channel upon SSE disconnect
func (l *Logger) unsubscribe(ch chan models.SSEEvent) {
	l.stateMutex.Lock()
	defer l.stateMutex.Unlock()

	if _, exists := l.clients[ch]; exists {
		delete(l.clients, ch)
		close(ch)
	}
}

// GetHistory returns the existing log history formatted for initial page connection
func (l *Logger) getHistory() string {
	l.stateMutex.Lock()
	defer l.stateMutex.Unlock()
	return strings.Join(l.logHistory, "\n")
}

// BroadcastLog formats a message, appends it to history, and broadcasts to SSE clients
func (l *Logger) BroadcastLog(message any) {
	var msg string
	switch v := message.(type) {
	case string:
		msg = v
	case error:
		msg = v.Error()
	default:
		msg = fmt.Sprint(v)
	}

	// Format time with 3-digit milliseconds
	msg = fmt.Sprintf("[%s] %s", strings.ReplaceAll(time.Now().Format("15:04:05.000"), ".", ":"), msg)

	l.stateMutex.Lock()

	// Maintain log memory history limit
	if l.MaxLogs > 0 && len(l.logHistory) >= l.MaxLogs {
		l.logHistory = l.logHistory[1:]
	}
	l.logHistory = append(l.logHistory, msg)

	// Format as standard SSE data frame
	sseFrame := models.SSEEvent{
		Data: msg,
	}

	// Broadcast frame to all connected SSE channels
	for ch := range l.clients {
		select {
		case ch <- sseFrame:
		default:
			// Non-blocking drop for slow clients
		}
	}
	l.stateMutex.Unlock()

	fmt.Println(msg)
}

// BroadcastConfig sends setting changes to all connected SSE clients
func (l *Logger) BroadcastConfig(ctx wsModels.Context) {
	data := Settings{}
	if err := models.ReadJsonBody(ctx, &data); err != nil {
		ctx.RespondJson(http.StatusBadRequest, err.Error())
		return
	}

	if data.MaxLogs == nil {
		return
	}

	l.stateMutex.Lock()
	l.MaxLogs = *data.MaxLogs

	// Trim history buffer immediately if maxLogs decreased
	if l.MaxLogs > 0 && len(l.logHistory) > l.MaxLogs {
		l.logHistory = l.logHistory[len(l.logHistory)-l.MaxLogs:]
	}

	sseFrame := models.SSEEvent{
		Event: "configChange",
		Data:  data,
	}

	for ch := range l.clients {
		select {
		case ch <- sseFrame:
		default:
		}
	}
	l.stateMutex.Unlock()

	ctx.RespondJson(http.StatusOK, map[string]string{"status": "ok"})
}
