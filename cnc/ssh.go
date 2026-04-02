package main

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConsole adapta un ssh.Channel para que implemente net.Conn
type SSHConsole struct {
	ssh.Channel
	remoteAddr     net.Addr
	localAddr      net.Addr
	terminalWidth  int32
	terminalHeight int32
}

func (c *SSHConsole) RemoteAddr() net.Addr { return c.remoteAddr }
func (c *SSHConsole) LocalAddr() net.Addr  { return c.localAddr }

func (c *SSHConsole) TerminalWidth() int {
	return int(atomic.LoadInt32(&c.terminalWidth))
}

func (c *SSHConsole) setTerminalSize(cols, rows int) {
	if cols > 0 {
		atomic.StoreInt32(&c.terminalWidth, int32(cols))
	}
	if rows > 0 {
		atomic.StoreInt32(&c.terminalHeight, int32(rows))
	}
}

// Stubs para los deadlines (no necesarios para nuestro uso, pero requeridos por net.Conn)
func (c *SSHConsole) SetDeadline(t time.Time) error      { return nil }
func (c *SSHConsole) SetReadDeadline(t time.Time) error  { return nil }
func (c *SSHConsole) SetWriteDeadline(t time.Time) error { return nil }

// startSSHServer inicia el servidor SSH en el puerto especificado
func startSSHServer(port string) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			user, err := FindUser(c.User())
			if err != nil || user == nil {
				return nil, fmt.Errorf("unknown user")
			}
			if user.Password != string(pass) {
				return nil, fmt.Errorf("incorrect password")
			}
			// Opcional: restringir SSH solo a administradores
			// if !user.Admin { return nil, fmt.Errorf("acceso solo para administradores") }
			return nil, nil
		},
	}

	// Generar clave privada RSA temporal en memoria
	privateKey, err := generateTempKey()
	if err != nil {
		fmt.Printf("[ssh] error generating key: %v\n", err)
		return
	}
	config.AddHostKey(privateKey)

	listener, err := net.Listen("tcp", port)
	if err != nil {
		fmt.Printf("[ssh] error listening on %s: %v\n", port, err)
		return
	}
	fmt.Printf("[ssh] SSH server listening on %s\n", port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go handleSSHConn(conn, config)
	}
}

// generateTempKey genera una clave RSA de 2048 bits y la convierte en ssh.Signer
func generateTempKey() (ssh.Signer, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, err
	}
	return signer, nil
}

// parsePtyRequestSize decodifica columnas/filas del payload SSH pty-req.
func parsePtyRequestSize(payload []byte) (cols int, rows int, ok bool) {
	if len(payload) < 4 {
		return 0, 0, false
	}

	termLen := int(binary.BigEndian.Uint32(payload[0:4]))
	offset := 4 + termLen
	if len(payload) < offset+16 {
		return 0, 0, false
	}

	cols = int(binary.BigEndian.Uint32(payload[offset : offset+4]))
	rows = int(binary.BigEndian.Uint32(payload[offset+4 : offset+8]))
	return cols, rows, true
}

// parseWindowChangeSize decodifica columnas/filas del payload SSH window-change.
func parseWindowChangeSize(payload []byte) (cols int, rows int, ok bool) {
	if len(payload) < 8 {
		return 0, 0, false
	}

	cols = int(binary.BigEndian.Uint32(payload[0:4]))
	rows = int(binary.BigEndian.Uint32(payload[4:8]))
	return cols, rows, true
}

// handleSSHConn maneja una conexion SSH entrante
func handleSSHConn(conn net.Conn, config *ssh.ServerConfig) {
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer sshConn.Close()
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unsupported channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		console := &SSHConsole{
			Channel:    channel,
			remoteAddr: conn.RemoteAddr(),
			localAddr:  conn.LocalAddr(),
		}

		// Process session requests
		go func(console *SSHConsole, requests <-chan *ssh.Request) {
			shellStarted := false

			for req := range requests {
				switch req.Type {
				case "pty-req":
					// Aceptar PTY para que el cliente SSH entre en modo interactivo
					// y no duplique el eco local de la terminal.
					if cols, rows, ok := parsePtyRequestSize(req.Payload); ok {
						console.setTerminalSize(cols, rows)
					}
					if req.WantReply {
						_ = req.Reply(true, nil)
					}
				case "window-change":
					if cols, rows, ok := parseWindowChangeSize(req.Payload); ok {
						console.setTerminalSize(cols, rows)
					}
					if req.WantReply {
						_ = req.Reply(true, nil)
					}
				case "shell":
					if shellStarted {
						if req.WantReply {
							_ = req.Reply(false, nil)
						}
						continue
					}
					shellStarted = true
					if req.WantReply {
						_ = req.Reply(true, nil)
					}
					// Run shell in its own goroutine so we keep handling
					// asynchronous requests like window-change while the user types.
					go func() {
						AdminSSH(console)
						_ = console.Close()
					}()
				case "exec":
					// Si recibimos un comando directo, lo rechazamos (solo queremos shell)
					if req.WantReply {
						_ = req.Reply(false, nil)
					}
				default:
					if req.WantReply {
						_ = req.Reply(false, nil)
					}
				}
			}
		}(console, requests)
	}
}
