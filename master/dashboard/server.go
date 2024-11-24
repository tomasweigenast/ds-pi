package dashboard

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"ds-pi.com/master/app"
	"ds-pi.com/master/stats"
	"github.com/gorilla/websocket"
)

//go:embed index.html
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
	// Start the stats updater in a goroutine
	go updateStats()

	// Start WebSocket server in a goroutine
	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/ws", handleWebSocket)
		log.Printf("Starting WebSocket server on :8080")
		if err := http.ListenAndServe(":8080", mux); err != nil {
			log.Fatal("WebSocket server error:", err)
		}
	}()

	// Start HTTP server for the dashboard
	mux := http.NewServeMux()

	// Serve the dashboard HTML file
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := content.ReadFile("index.html")
		if err != nil {
			http.Error(w, "Failed to read dashboard", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	})

	log.Printf("Starting HTTP server on :80")
	if err := http.ListenAndServe(":80", mux); err != nil {
		log.Fatal("HTTP server error:", err)
	}
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
	for {
		// Read message (required to keep connection alive)
		if _, _, err := conn.ReadMessage(); err != nil {
			log.Printf("WebSocket read error: %v", err)
			return
		}
	}
}

func broadcastStats(stats stats.Stats) {
	activeConnections.Range(func(key, value interface{}) bool {
		conn := value.(*websocket.Conn)
		if err := conn.WriteJSON(stats); err != nil {
			log.Printf("Failed to send stats to client: %v", err)
			activeConnections.Delete(key)
			conn.Close()
		} else {
			log.Printf("Stats sent to %s", key)
		}
		return true
	})
}

func updateStats() {
	// Simulate some workers for demo purposes
	// workers := []Worker{
	// 	{ID: "worker-1", Active: true, LastPing: time.Now(), LastJob: "Processing batch 1"},
	// 	{ID: "worker-2", Active: true, LastPing: time.Now(), LastJob: "Processing batch 2"},
	// }

	ticker := time.NewTicker(1 * time.Second)
	ticker2 := time.NewTicker(10 * time.Second)
	// piValue := 3.0

	go func() {
		for range ticker.C {
			statsMutex.Lock()
			serverStats := app.Stats()
			broadcastStats(stats.Stats{
				Server: &serverStats,
			})
			statsMutex.Unlock()
		}
	}()

	go func() {
		for range ticker2.C {
			statsMutex.Lock()
			piStats := app.PIStats()
			broadcastStats(stats.Stats{
				PI: &piStats,
			})
			statsMutex.Unlock()
		}
	}()
}
