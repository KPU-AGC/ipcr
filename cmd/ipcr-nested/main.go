// cmd/ipcr-nested/main.go  (REPLACE)
package main

import (
	"ipcr/internal/appshell"
	"ipcr/internal/nestedapp"
)

func main() { appshell.Main(nestedapp.RunContext) }
