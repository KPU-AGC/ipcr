// cmd/ipcr/main.go  (REPLACE)
package main

import (
	"ipcr/internal/app"
	"ipcr/internal/appshell"
)

func main() { appshell.Main(app.RunContext) }
