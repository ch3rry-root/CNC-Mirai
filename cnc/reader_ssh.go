package main

import (
	"net"
	"strings"
)

// readSSHLine es un lector simple para sesiones interactivas SSH.
// Soporta:
// - Backspace
// - Flechas izquierda/derecha (movimiento del cursor)
// - Flechas arriba/abajo (historial)
// - Entrada normal con eco
func readSSHLine(conn net.Conn, prompt, blocked string, maxLen int, history []string) (string, error) {
	if _, err := conn.Write([]byte(prompt)); err != nil {
		return "", err
	}
	message := make([]byte, 0, maxLen)
	pos := len(history)
	cursor := 0
	widthPerChar := 1
	if blocked != "" {
		widthPerChar = len(blocked)
	}

	displayLine := func(line []byte) string {
		if blocked == "" {
			return string(line)
		}
		if len(line) == 0 {
			return ""
		}
		return strings.Repeat(blocked, len(line))
	}

	redraw := func(line []byte) error {
		// Limpiar la lÃ­nea actual y redibujar con el prompt y el contenido
		_, err := conn.Write([]byte("\r\x1b[2K" + prompt + displayLine(line)))
		return err
	}

	moveCursorTo := func(line []byte, cursorPos int) error {
		if cursorPos < 0 {
			cursorPos = 0
		}
		if cursorPos > len(line) {
			cursorPos = len(line)
		}
		if err := redraw(line); err != nil {
			return err
		}
		moveLeft := (len(line) - cursorPos) * widthPerChar
		if moveLeft <= 0 {
			return nil
		}
		_, err := conn.Write([]byte(strings.Repeat("\x1b[D", moveLeft)))
		return err
	}

	handleHistory := func(newPos int) error {
		pos = newPos
		if pos < 0 {
			pos = 0
		}
		if pos >= len(history) {
			pos = len(history)
			message = message[:0]
			cursor = 0
			return redraw(message)
		}
		message = []byte(history[pos])
		cursor = len(message)
		return redraw(message)
	}

	for {
		// Leer un byte
		var b [1]byte
		n, err := conn.Read(b[:])
		if err != nil || n == 0 {
			return "", err
		}
		ch := b[0]

		// Secuencias de escape para flechas
		if ch == 27 { // ESC
			seq, err := readEscapeSequence(conn)
			if err != nil {
				continue
			}
			switch seq {
			case "[A": // Up
				if pos > 0 {
					handleHistory(pos - 1)
				}
			case "[B": // Down
				if pos < len(history) {
					handleHistory(pos + 1)
				}
			case "[D": // Left
				if cursor > 0 {
					cursor--
					conn.Write([]byte("\x1b[D"))
				}
			case "[C": // Right
				if cursor < len(message) {
					cursor++
					conn.Write([]byte("\x1b[C"))
				}
			}
			continue
		}

		switch ch {
		case 8, 127: // Backspace
			if cursor > 0 {
				copy(message[cursor-1:], message[cursor:])
				message = message[:len(message)-1]
				cursor--
				moveCursorTo(message, cursor)
			}
		case 13: // Enter
			if len(message) == 0 {
				continue
			}
			conn.Write([]byte("\r\n"))
			return strings.TrimSpace(string(message)), nil
		default:
			if ch < 32 { // Ignorar otros controles
				continue
			}
			if len(message)+1 > maxLen {
				conn.Write([]byte("\x07")) // Beep
				continue
			}
			// Insertar carÃ¡cter en la posiciÃ³n actual
			if cursor == len(message) {
				message = append(message, ch)
				if blocked == "" {
					conn.Write([]byte{ch})
				} else {
					conn.Write([]byte(blocked))
				}
				cursor++
				pos = len(history)
			} else {
				message = append(message, 0)
				copy(message[cursor+1:], message[cursor:])
				message[cursor] = ch
				cursor++
				moveCursorTo(message, cursor)
				pos = len(history)
			}
		}
	}
}

// readEscapeSequence lee una secuencia de escape ANSI (despuÃ©s de ESC)
func readEscapeSequence(conn net.Conn) (string, error) {
	var buf [2]byte
	n, err := conn.Read(buf[:1])
	if err != nil || n == 0 {
		return "", err
	}
	if buf[0] == '[' {
		// Puede ser [A, [B, [C, [D] o secuencias mÃ¡s largas (para flechas basta)
		n, err = conn.Read(buf[:1])
		if err != nil || n == 0 {
			return "", err
		}
		return "[" + string(buf[0]), nil
	} else if buf[0] == 'O' {
		n, err = conn.Read(buf[:1])
		if err != nil || n == 0 {
			return "", err
		}
		return "O" + string(buf[0]), nil
	}
	return "", nil
}
