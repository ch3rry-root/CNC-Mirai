package main

import (
	"net"
	"strings"
)

// Wraps the read function
func Read(conn net.Conn, prompt, blocked string, maximumLen int) (string, error) {
	return read(conn, prompt, blocked, maximumLen, make([]string, 0))
}

// Wraps the read function
func ReadWithHistory(conn net.Conn, prompt, blocked string, maximumLen int, history []string) (string, error) {
	return read(conn, prompt, blocked, maximumLen, history)
}

// Read will act as the reader for taking inputs from master connections
func read(conn net.Conn, prompt, blocked string, maximumLen int, history []string) (string, error) {
	if _, err := conn.Write([]byte(prompt)); err != nil {
		return "", err
	}

	const cr = "\r\x00" // TELNET NVT: CR must be followed by NUL to avoid LF
	message := make([]byte, 0, maximumLen)
	if _, err := conn.Write([]byte{255, 251, 1, 255, 251, 3, 255, 252, 34}); err != nil {
		return "", err
	}

	pos := len(history)
	cursor := 0
	widthPerChar := 1
	if blocked != "" {
		widthPerChar = len(blocked)
	}

	readByte := func() (byte, error) {
		var b [1]byte
		for {
			n, err := conn.Read(b[:])
			if err != nil {
				return 0, err
			}
			if n == 1 {
				return b[0], nil
			}
		}
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
		_, err := conn.Write([]byte(cr + "\x1b[2K" + prompt + displayLine(line)))
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

	discardSubnegotiation := func() error {
		for {
			b, err := readByte()
			if err != nil {
				return err
			}
			if b == 255 {
				next, err := readByte()
				if err != nil {
					return err
				}
				if next == 240 { // SE
					return nil
				}
			}
		}
	}

	for {
		b, err := readByte()
		if err != nil {
			return "", err
		}

		if b == 255 { // IAC
			cmd, err := readByte()
			if err != nil {
				return "", err
			}
			switch cmd {
			case 255:
				continue
			case 250: // SB
				if err := discardSubnegotiation(); err != nil {
					return "", err
				}
				continue
			case 251, 252, 253, 254:
				if _, err := readByte(); err != nil {
					return "", err
				}
				continue
			default:
				continue
			}
		}

		if b == 27 { // ESC
			b1, err := readByte()
			if err != nil {
				return "", err
			}
			if b1 == '[' {
				var final byte
				for {
					b2, err := readByte()
					if err != nil {
						return "", err
					}
					final = b2
					if final >= 0x40 && final <= 0x7E {
						break
					}
				}
				switch final {
				case 'A':
					if pos <= 0 {
						continue
					}
					if err := handleHistory(pos - 1); err != nil {
						return "", err
					}
				case 'B':
					if err := handleHistory(pos + 1); err != nil {
						return "", err
					}
				case 'D':
					if cursor <= 0 {
						continue
					}
					cursor--
					if _, err := conn.Write([]byte(strings.Repeat("\x1b[D", widthPerChar))); err != nil {
						return "", err
					}
				case 'C':
					if cursor >= len(message) {
						continue
					}
					cursor++
					if _, err := conn.Write([]byte(strings.Repeat("\x1b[C", widthPerChar))); err != nil {
						return "", err
					}
				}
				continue
			}
			if b1 == 'O' {
				b2, err := readByte()
				if err != nil {
					return "", err
				}
				switch b2 {
				case 'A':
					if pos <= 0 {
						continue
					}
					if err := handleHistory(pos - 1); err != nil {
						return "", err
					}
				case 'B':
					if err := handleHistory(pos + 1); err != nil {
						return "", err
					}
				case 'D':
					if cursor <= 0 {
						continue
					}
					cursor--
					if _, err := conn.Write([]byte(strings.Repeat("\x1b[D", widthPerChar))); err != nil {
						return "", err
					}
				case 'C':
					if cursor >= len(message) {
						continue
					}
					cursor++
					if _, err := conn.Write([]byte(strings.Repeat("\x1b[C", widthPerChar))); err != nil {
						return "", err
					}
				}
				continue
			}
			continue
		}

		switch b {
		case 0, 10:
			continue
		case 8, 127:
			if cursor > 0 {
				copy(message[cursor-1:], message[cursor:])
				message = message[:len(message)-1] // Eliminar del buffer
				cursor--
				if err := moveCursorTo(message, cursor); err != nil {
					return "", err
				}
			}
			continue
		case 13:
			if len(message) <= 0 {
				continue
			}
			if _, err := conn.Write([]byte("\r\n")); err != nil {
				return "", err
			}
			return strings.TrimSpace(string(message)), nil // Limpiar espacios innecesarios
		default:
			if b < 32 {
				continue
			}
			if len(message)+1 > maximumLen {
				conn.Write([]byte("\x07")) // Emitir sonido si se excede el tamaño
				continue
			}

			if cursor == len(message) {
				var write string
				if blocked == "" {
					write = string([]byte{b})
				} else {
					write = blocked
				}

				if _, err := conn.Write([]byte(write)); err != nil {
					return "", err
				}

				message = append(message, b)
				cursor++
				pos = len(history)
				continue
			}

			message = append(message, 0)
			copy(message[cursor+1:], message[cursor:])
			message[cursor] = b
			cursor++
			if err := moveCursorTo(message, cursor); err != nil {
				return "", err
			}
			pos = len(history)
		}
	}
}
