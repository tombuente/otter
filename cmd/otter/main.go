package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/tombuente/otter/internal/provider"
)

func main() {
	ctx := context.Background()

	// TODO for testing
	data, err := os.Open("./test.toml")
	if err != nil {
		fmt.Println(err)
	}

	buffer, err := io.ReadAll(data)
	if err != nil {
		slog.Error("Unable to read from config file", "error", err)
	}

	provider, err := provider.NewProvider(buffer)
	if err != nil {
		slog.Error("Unable to create new provider", "error", err)
	}
	defer provider.Close()

	// TODO for testing
	if err = provider.Up(ctx); err != nil {
		fmt.Println(err)
	}
}
