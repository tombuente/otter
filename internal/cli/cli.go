package cli

import (
	"fmt"

	providers "github.com/tombuente/otter/internal/provider"
	"github.com/urfave/cli/v2"
)

func New() *cli.App {
	return &cli.App{
		Name:  "otter",
		Usage: "Manage containers",
		Commands: []*cli.Command{
			{
				Name:  "up",
				Usage: "Create containers",

				Action: func(ctx *cli.Context) error {
					provider, err := providers.NewFromConfig(ctx.Args().First())
					if err != nil {
						return fmt.Errorf("unable to create provider from config file: %w", err)
					}
					defer provider.Close()

					provider.Up(ctx.Context)

					return nil
				},
			},
		},
	}
}
