package dashboard

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"ds-pi.com/master/app"
	"ds-pi.com/master/shared"
	"ds-pi.com/master/stats"
	"github.com/gorilla/websocket"
)

//go:embed index.html pi.html
var content embed.FS

var (
	upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Global state
	activeConnections sync.Map
	currentStats      stats.Stats
	statsMutex        sync.Mutex
)

func Start() {
	// Start WebSocket server in a goroutine
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/ws", handleWebSocket)
		mux.HandleFunc("/pi", handlePI)
		mux.HandleFunc("/", handleIndex)
		log.Printf("Starting HTTP server on :80")
		if err := http.ListenAndServe(":80", mux); err != nil {
			log.Fatal("HTTP server error:", err)
		}
	}()

	// Start the stats updater in a goroutine
	go updateStats()
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	data, err := content.ReadFile("index.html")
	if err != nil {
		http.Error(w, "Failed to read dashboard", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")

	ip, err := shared.GetIPv4()
	if err == nil {
		dataStr := strings.ReplaceAll(string(data), "{{ip}}", ip.String())
		data = []byte(dataStr)
	}
	w.Write(data)
}

func handlePI(w http.ResponseWriter, r *http.Request) {
	pi := app.PIStats().PI
	data, err := content.ReadFile("pi.html")
	if err != nil {
		http.Error(w, "Failed to read dashboard", http.StatusInternalServerError)
		return
	}

	stringContent := string(data)
	stringContent = strings.ReplaceAll(stringContent, "{{pi}}", pi)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(stringContent))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	// Generate a unique ID for this connection
	connID := fmt.Sprintf("%p", conn)
	activeConnections.Store(connID, conn)
	defer activeConnections.Delete(connID)

	// Keep the connection alive and handle any incoming messages
	select {}
}

func broadcastStats(stats stats.Stats) {
	statsMutex.Lock()
	defer statsMutex.Unlock()

	activeConnections.Range(func(key, value interface{}) bool {
		conn := value.(*websocket.Conn)
		if err := conn.WriteJSON(stats); err != nil {
			log.Printf("Failed to send stats to client: %v", err)
			activeConnections.Delete(key)
			conn.Close()
		}
		return true
	})
}

func updateStats() {

	ticker := time.NewTicker(1 * time.Second)
	ticker2 := time.NewTicker(10 * time.Second)
	// piValue := 3.0

	go func() {
		for range ticker.C {
			serverStats := app.Stats()
			broadcastStats(stats.Stats{
				Server: &serverStats,
			})
		}
	}()

	go func() {
		for range ticker2.C {
			piStats := app.PIStats()
			broadcastStats(stats.Stats{
				PI: &piStats,
			})
		}
	}()
}
