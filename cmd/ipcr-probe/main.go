// cmd/ipcr-probe/main.go  (REPLACE)
package main

import (
	"ipcr/internal/appshell"
	"ipcr/internal/probeapp"
)

func main() { appshell.Main(probeapp.RunContext) }
