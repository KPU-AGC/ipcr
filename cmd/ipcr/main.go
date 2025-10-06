// cmd/ipcr/main.go 
package main

import (
	"ipcr/internal/app"
	"ipcr/internal/appshell"
)

func main() { appshell.Main(app.RunContext) }
