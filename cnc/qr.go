package main

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/mdp/qrterminal/v3"
)

const mfaIssuer = "CNC Mirai"

// buildMFAProvisioningURI returns a standard otpauth URI for TOTP apps.
func buildMFAProvisioningURI(username, secret string) string {
	label := url.PathEscape(fmt.Sprintf("%s:%s", mfaIssuer, username))
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", mfaIssuer)
	// digits/period/algorithm are defaults in most authenticator apps
	return fmt.Sprintf("otpauth://totp/%s?%s", label, q.Encode())
}

// renderMFAASCIIQR genera el QR usando la librería qrterminal
func renderMFAASCIIQR(provisioningURI string, terminalWidth int) (string, error) {
	var buf bytes.Buffer

	// Configuramos qrterminal para la máxima compatibilidad
	config := qrterminal.Config{
		Level:     qrterminal.L, // Low error correction = QR más pequeño (ideal para terminales 80x24)
		Writer:    &buf,
		BlackChar: qrterminal.BLACK, // Usa colores ANSI universales (fondo negro + dos espacios)
		WhiteChar: qrterminal.WHITE, // Usa colores ANSI universales (fondo blanco + dos espacios)
		QuietZone: 2,                // Margen blanco para que la cámara enfoque bien
	}

	qrterminal.GenerateWithConfig(provisioningURI, config)

	// FIX CRÍTICO PARA SSH:
	// Convertimos los saltos de línea de Unix (\n) al formato requerido por las PTY de SSH (\r\n).
	out := strings.ReplaceAll(buf.String(), "\n", "\r\n")

	// OPCIONAL: Si el resultado no tiene un salto de línea al inicio y quieres separarlo del texto anterior
	out = "\r\n" + out

	return out, nil
}
