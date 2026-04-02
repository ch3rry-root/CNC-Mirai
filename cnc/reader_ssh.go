package main

import (
	"net"
	"strings"
)

// readSSHLine is a simple interactive SSH line reader.
// Supports:
// - Backspace
// - Left/Right arrows (cursor movement)
// - Up/Down arrows (history)
// - Normal input with echo
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
		var b [1]byte
		n, err := conn.Read(b[:])
		if err != nil || n == 0 {
			return "", err
		}
		ch := b[0]

		if ch == 27 { // ESC
			seq, err := readEscapeSequence(conn)
			if err != nil {
				continue
			}
			switch {
			case strings.HasSuffix(seq, "A"): // Up
				if pos > 0 {
					_ = handleHistory(pos - 1)
				}
			case strings.HasSuffix(seq, "B"): // Down
				if pos < len(history) {
					_ = handleHistory(pos + 1)
				}
			case strings.HasSuffix(seq, "D"): // Left
				if cursor > 0 {
					cursor--
					_, _ = conn.Write([]byte("\x1b[D"))
				}
			case strings.HasSuffix(seq, "C"): // Right
				if cursor < len(message) {
					cursor++
					_, _ = conn.Write([]byte("\x1b[C"))
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
				_ = moveCursorTo(message, cursor)
			}
		case 13: // Enter
			if len(message) == 0 {
				continue
			}
			_, _ = conn.Write([]byte("\r\n"))
			return strings.TrimSpace(string(message)), nil
		default:
			if ch < 32 {
				continue
			}
			if len(message)+1 > maxLen {
				_, _ = conn.Write([]byte("\x07"))
				continue
			}

			if cursor == len(message) {
				message = append(message, ch)
				if blocked == "" {
					_, _ = conn.Write([]byte{ch})
				} else {
					_, _ = conn.Write([]byte(blocked))
				}
				cursor++
				pos = len(history)
			} else {
				message = append(message, 0)
				copy(message[cursor+1:], message[cursor:])
				message[cursor] = ch
				cursor++
				_ = moveCursorTo(message, cursor)
				pos = len(history)
			}
		}
	}
}

func isANSISequenceFinalByte(b byte) bool {
	return b >= 0x40 && b <= 0x7E
}

// readEscapeSequence reads an ANSI escape sequence after ESC.
func readEscapeSequence(conn net.Conn) (string, error) {
	var b [1]byte
	n, err := conn.Read(b[:1])
	if err != nil || n == 0 {
		return "", err
	}

	switch b[0] {
	case '[', 'O':
		seq := []byte{b[0]}
		for i := 0; i < 64; i++ {
			n, err = conn.Read(b[:1])
			if err != nil || n == 0 {
				return string(seq), err
			}
			seq = append(seq, b[0])
			if isANSISequenceFinalByte(b[0]) {
				break
			}
		}
		return string(seq), nil
	case ']':
		seq := []byte{b[0]}
		for i := 0; i < 256; i++ {
			n, err = conn.Read(b[:1])
			if err != nil || n == 0 {
				return string(seq), err
			}
			seq = append(seq, b[0])
			if b[0] == 0x07 {
				break
			}
			if b[0] == 0x1b {
				n, err = conn.Read(b[:1])
				if err != nil || n == 0 {
					return string(seq), err
				}
				seq = append(seq, b[0])
				if b[0] == '\\' {
					break
				}
			}
		}
		return string(seq), nil
	default:
		return string(b[0]), nil
	}
}
