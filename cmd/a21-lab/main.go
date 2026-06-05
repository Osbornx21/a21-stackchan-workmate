package main

import (
	"os"

	"a21.local/a21/internal/app"
)

func main() {
	os.Exit(app.RunLab(os.Args[1:], os.Stdout, os.Stderr))
}
