package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/joona/cockcli/internal/commands"
	"github.com/joona/cockcli/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("unable to load config: %v", err)
	}

	app := &cli.App{
		Name:    "cockcli",
		Usage:   "Interact with Cockpit CMS collections via API",
		Version: "0.0.1",
		Authors: []*cli.Author{{Name: "Joona Kulmala", Email: "jmkulmala@gmail.com"}},
		Metadata: map[string]any{
			"config": cfg, // pass to sub‑commands
		},
		Flags: []cli.Flag{
			&cli.StringFlag{ // instance alias *required*
				Name:     "instance",
				Aliases:  []string{"i"},
				Usage:    "Instance alias defined in config file (required)",
				Required: true,
			},
			&cli.StringFlag{ // optional override URL
				Name:  "url",
				Usage: "Override Cockpit base URL (takes precedence over instance config)",
			},
			&cli.StringFlag{ // optional override token
				Name:    "token",
				Aliases: []string{"t"},
				Usage:   "Override API token (takes precedence over config)",
				EnvVars: []string{"COCKPIT_TOKEN"},
			},
		},
		Commands: []*cli.Command{
			commands.ListCmd(),
			commands.GetCmd(),
			commands.UpdateCmd(),
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
