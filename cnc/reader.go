package main

import (
	"fmt"
	"net"
	"strings"
)

func Read(conn net.Conn, prompt, blocked string, maximumLen int) (string, error) {
	return read(conn, prompt, blocked, maximumLen, make([]string, 0))
}

func ReadWithHistory(conn net.Conn, prompt, blocked string, maximumLen int, history []string) (string, error) {
	return read(conn, prompt, blocked, maximumLen, history)
}

func read(conn net.Conn, prompt, blocked string, maximumLen int, history []string) (string, error) {

	conn.Write([]byte(prompt))

	message := make([]string, 0)
	pos := len(history)

	for {
		buf := make([]byte, 1)
		_, err := conn.Read(buf)
		if err != nil {
			return "", err
		}

		switch buf[0] {

		case 127, 8: // Backspace
			if len(message) > 0 {
				message = message[:len(message)-1]
				conn.Write([]byte("\b \b"))
			}

		case 13: // Enter
			if len(message) == 0 {
				continue
			}

			conn.Write([]byte("\r\n"))
			return strings.Join(message, ""), nil

		case 27: // Flechas
			buffer := make([]byte, 2)
			conn.Read(buffer)

			if buffer[0] != 91 {
				continue
			}

			switch buffer[1] {

			case 65: // ↑ historial arriba
				if pos <= 0 {
					continue
				}

				pos--
				conn.Write([]byte(fmt.Sprintf("\r\033[2K\r%s%s", prompt, history[pos])))
				message = strings.Split(history[pos], "")

			case 66: // ↓ historial abajo
				if pos+1 >= len(history) {
					continue
				}

				pos++
				conn.Write([]byte(fmt.Sprintf("\r\033[2K\r%s%s", prompt, history[pos])))
				message = strings.Split(history[pos], "")
			}

		default:

			if len(message)+1 > maximumLen {
				conn.Write([]byte("\x07"))
				continue
			}

			char := string(buf[0])

			write := char
			if len(blocked) > 0 {
				write = blocked
			}

			conn.Write([]byte(write))

			message = append(message, char)

			// salir del historial cuando escribes algo nuevo
			pos = len(history)
		}
	}
}
