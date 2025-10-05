// cmd/ipcr-thermo/main.go
package main

import (
	"ipcr/internal/appshell"
	"ipcr/internal/thermoapp"
)

func main() { appshell.Main(thermoapp.RunContext) }
