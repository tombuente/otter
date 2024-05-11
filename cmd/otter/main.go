package main

import (
	"log/slog"
	"os"

	"github.com/tombuente/otter/internal/cli"
)

func main() {
	if err := cli.New().Run(os.Args); err != nil {
		slog.Error(err.Error())
		return
	}
}
