// cmd/ipcr-multiplex/main.go  (REPLACE)
package main

import (
	"ipcr/internal/appshell"
	"ipcr/internal/multiplexapp"
)

func main() { appshell.Main(multiplexapp.RunContext) }
