package main

import (
	"bytes"
	"fmt"
	"os"

	"ipcress-go/internal/app"
)

func main() {
	var out, errBuf bytes.Buffer
	code := app.Run(os.Args[1:], &out, &errBuf)

	if out.Len() > 0 {
		fmt.Print(out.String())
	}
	if errBuf.Len() > 0 {
		fmt.Fprint(os.Stderr, errBuf.String())
	}
	os.Exit(code)
}
