package main

import (
	"os"

	"github.com/aredgwell/athena/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		os.Exit(cli.ExitCodeForError(err))
	}
}
