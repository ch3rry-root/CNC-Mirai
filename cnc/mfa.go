package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"
)

const (
	mfaCodeDigits       = 6
	mfaPeriodSeconds    = 30
	mfaMaxAttemptRounds = 3
	mfaSecretBytes      = 10
)

func generateMFASecret() (string, error) {
	raw := make([]byte, mfaSecretBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(raw), nil
}

func formatMFASecretForDisplay(secret string) string {
	clean := strings.ToUpper(strings.TrimSpace(secret))
	if clean == "" {
		return clean
	}

	var out strings.Builder
	for i := 0; i < len(clean); i++ {
		if i > 0 && i%4 == 0 {
			out.WriteByte(' ')
		}
		out.WriteByte(clean[i])
	}
	return out.String()
}

func normalizeMFACode(input string) string {
	cleaned := strings.TrimSpace(input)
	cleaned = strings.ReplaceAll(cleaned, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")
	return cleaned
}

func decodeMFASecret(secret string) ([]byte, error) {
	normalized := strings.ToUpper(strings.TrimSpace(secret))
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, "-", "")
	if normalized == "" {
		return nil, errors.New("empty mfa secret")
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(normalized)
}

func generateTOTPCode(secret string, t time.Time) (string, error) {
	key, err := decodeMFASecret(secret)
	if err != nil {
		return "", err
	}

	counter := uint64(t.Unix() / mfaPeriodSeconds)
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)

	h := hmac.New(sha1.New, key)
	_, _ = h.Write(buf[:])
	sum := h.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	binaryCode := (uint32(sum[offset])&0x7f)<<24 |
		(uint32(sum[offset+1])&0xff)<<16 |
		(uint32(sum[offset+2])&0xff)<<8 |
		(uint32(sum[offset+3]) & 0xff)

	code := binaryCode % 1000000
	return fmt.Sprintf("%06d", code), nil
}

func validateTOTPCode(secret, code string, now time.Time) bool {
	input := normalizeMFACode(code)
	if len(input) != mfaCodeDigits {
		return false
	}

	for skew := -1; skew <= 1; skew++ {
		t := now.Add(time.Duration(skew*mfaPeriodSeconds) * time.Second)
		expected, err := generateTOTPCode(secret, t)
		if err != nil {
			return false
		}
		if input == expected {
			return true
		}
	}

	return false
}

func requestMFACode(conn net.Conn, prompt string) (string, error) {
	return readSSHLine(conn, prompt, "", 12, []string{})
}

func setupUserMFA(conn net.Conn, user *User) error {
	secret, err := generateMFASecret()
	if err != nil {
		return err
	}

	conn.Write([]byte("\r\n" + ansiPrompt + "MFA setup required for your account." + ansiReset + "\r\n"))
	conn.Write([]byte(ansiSeparator + "Open your Authenticator app and choose manual key setup (TOTP)." + ansiReset + "\r\n"))
	conn.Write([]byte(ansiSeparator + "Issuer" + ansiReset + ": " + ansiCommands + mfaIssuer + ansiReset + "\r\n"))
	conn.Write([]byte(ansiSeparator + "Account" + ansiReset + ": " + ansiCommands + user.Username + ansiReset + "\r\n"))
	conn.Write([]byte(ansiSeparator + "Manual key" + ansiReset + ": " + ansiNumbers + formatMFASecretForDisplay(secret) + ansiReset + "\r\n"))
	conn.Write([]byte(ansiSeparator + "Enter the current 6-digit code to finish setup." + ansiReset + "\r\n\r\n"))

	prompt := fmt.Sprintf("%sMFA code%s %s->%s %s", ansiPrompt, ansiReset, ansiSuccess, ansiReset, ansiCommands)
	for attempt := 1; attempt <= mfaMaxAttemptRounds; attempt++ {
		code, err := requestMFACode(conn, prompt)
		if err != nil {
			return err
		}
		if validateTOTPCode(secret, code, time.Now()) {
			if err := ModifyField(user, "mfa_secret", secret); err != nil {
				return err
			}
			if err := ModifyField(user, "mfa", true); err != nil {
				return err
			}
			user.MFASecret = secret
			user.MFA = true
			conn.Write([]byte(ansiSuccess + "MFA configured." + ansiReset + "\r\n"))
			return nil
		}
		remaining := mfaMaxAttemptRounds - attempt
		if remaining > 0 {
			conn.Write([]byte(fmt.Sprintf("%sInvalid code.%s %s%d%s attempt(s) left.\r\n", ansiError, ansiReset, ansiNumbers, remaining, ansiReset)))
		}
	}

	return errors.New("mfa setup failed")
}

func verifyUserMFA(conn net.Conn, user *User) error {
	if user == nil {
		return nil
	}
	if strings.EqualFold(user.Username, "root") {
		return nil
	}
	if !user.MFA {
		return nil
	}

	if strings.TrimSpace(user.MFASecret) == "" {
		if err := setupUserMFA(conn, user); err != nil {
			return err
		}
		return nil
	}

	prompt := fmt.Sprintf("%sMFA code%s %s->%s %s", ansiPrompt, ansiReset, ansiSuccess, ansiReset, ansiCommands)
	for attempt := 1; attempt <= mfaMaxAttemptRounds; attempt++ {
		code, err := requestMFACode(conn, prompt)
		if err != nil {
			return err
		}
		if validateTOTPCode(user.MFASecret, code, time.Now()) {
			return nil
		}

		remaining := mfaMaxAttemptRounds - attempt
		if remaining > 0 {
			conn.Write([]byte(fmt.Sprintf("%sInvalid MFA code.%s %s%d%s attempt(s) left.\r\n", ansiError, ansiReset, ansiNumbers, remaining, ansiReset)))
		}
	}

	return errors.New("mfa verification failed")
}
