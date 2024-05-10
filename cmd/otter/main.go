package main

import (
	"log/slog"
	"os"

	"github.com/tombuente/otter/internal/cli"
)

func main() {
	// ctx := context.Background()

	// TODO for testing
	// data, err := os.Open("./test.toml")
	// if err != nil {
	// 	fmt.Println(err)
	// }

	// buffer, err := io.ReadAll(data)
	// if err != nil {
	// 	slog.Error("Unable to read from config file", "error", err)
	// }

	// provider, err := provider.NewProvider(buffer)
	// if err != nil {
	// 	slog.Error("Unable to create new provider", "error", err)
	// }
	// defer provider.Close()

	// TODO for testing
	// if err = provider.Up(ctx); err != nil {
	// 	fmt.Println(err)
	// }

	if err := cli.New().Run(os.Args); err != nil {
		slog.Error(err.Error())
		return
	}
}
