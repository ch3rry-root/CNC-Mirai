package main

import (
	"net"
)

const indent = "  "

// writeLine escribe una línea con dos espacios, viñeta y texto básico (sin color interno)
func writeLine(conn net.Conn, text string) {
	line := indent + ansiSystem + "•" + ansiReset + " " + ansiCommands + text + ansiReset
	conn.Write([]byte(line + "\r\n"))
}

// writeKeyValue escribe "clave: valor" con formato estándar
func writeKeyValue(conn net.Conn, key string, value string) {
	line := indent + ansiSystem + "•" + ansiReset + " " +
		ansiCommands + key + ansiReset + ansiSystem + ":" + ansiReset + "\t" +
		ansiCommands + value + ansiReset
	conn.Write([]byte(line + "\r\n"))
}

// writeKeyValueColor permite especificar colores para clave y valor
func writeKeyValueColor(conn net.Conn, key string, keyColor string, value string, valueColor string) {
	line := indent + ansiSystem + "•" + ansiReset + " " +
		keyColor + key + ansiReset + ansiSystem + ":" + ansiReset + "\t" +
		valueColor + value + ansiReset
	conn.Write([]byte(line + "\r\n"))
}

// writeHeader escribe una línea de encabezado sin colon (ej. "Bots by country:")
func writeHeader(conn net.Conn, header string) {
	line := indent + ansiSystem + "•" + ansiReset + " " + ansiCommands + header + ansiReset
	conn.Write([]byte(line + "\r\n"))
}
