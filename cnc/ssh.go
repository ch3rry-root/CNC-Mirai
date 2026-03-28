package main

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// SSHConsole adapta un ssh.Channel para que implemente net.Conn
type SSHConsole struct {
	ssh.Channel
	remoteAddr net.Addr
	localAddr  net.Addr
}

func (c *SSHConsole) RemoteAddr() net.Addr { return c.remoteAddr }
func (c *SSHConsole) LocalAddr() net.Addr  { return c.localAddr }

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

// handleSSHConn maneja una conexión SSH entrante
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
		// Process session requests
		go func() {
			for req := range requests {
				switch req.Type {
				case "pty-req":
					// Aceptar PTY para que el cliente SSH entre en modo interactivo
					// y no duplique el eco local de la terminal.
					req.Reply(true, nil)
				case "window-change":
					// Ignoramos el tamaÃ±o por ahora, pero confirmamos el request.
					req.Reply(true, nil)
				case "shell":
					req.Reply(true, nil)
					// Crear un wrapper que implemente net.Conn
					console := &SSHConsole{
						Channel:    channel,
						remoteAddr: conn.RemoteAddr(),
						localAddr:  conn.LocalAddr(),
					}
					// Llamar a la función Admin existente (que espera net.Conn)
					AdminSSH(console)
					return
				case "exec":
					// Si recibimos un comando directo, lo rechazamos (solo queremos shell)
					req.Reply(false, nil)
				default:
					req.Reply(false, nil)
				}
			}
		}()
	}
}
