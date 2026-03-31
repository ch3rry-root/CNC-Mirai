package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	workers    = make(map[*websocket.Conn]string)
	workersMux sync.RWMutex
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// startWorkerServer inicia el servidor WebSocket para los workers L7.
func startWorkerServer() {
	http.HandleFunc("/worker", handleWorker)
	log.Printf("[Worker] WebSocket server listening on :1995/worker")
	if err := http.ListenAndServe(":1995", nil); err != nil {
		log.Fatal("[Worker] Server error:", err)
	}
}

func handleWorker(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[Worker] Upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Leer mensaje de registro
	var msg map[string]interface{}
	if err := conn.ReadJSON(&msg); err != nil {
		log.Printf("[Worker] Registration error: %v", err)
		return
	}
	name, ok := msg["name"].(string)
	if !ok {
		conn.WriteJSON(map[string]string{"error": "missing name"})
		return
	}

	workersMux.Lock()
	workers[conn] = name
	workersMux.Unlock()
	log.Printf("[Worker] %s connected", name)

	// Mantener la conexión viva (simplemente esperar)
	for {
		if _, _, err := conn.NextReader(); err != nil {
			break
		}
	}

	workersMux.Lock()
	delete(workers, conn)
	workersMux.Unlock()
	log.Printf("[Worker] %s disconnected", name)
}

// SendToWorkers envía un comando a todos los workers conectados.
func SendToWorkers(cmd map[string]interface{}) {
	workersMux.RLock()
	defer workersMux.RUnlock()
	for conn := range workers {
		if err := conn.WriteJSON(cmd); err != nil {
			conn.Close()
		}
	}
}
