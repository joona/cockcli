package commands

import (
	"errors"

	"github.com/urfave/cli/v2"

	"github.com/joona/cockcli/internal/client"
	"github.com/joona/cockcli/internal/config"
)

// getClient resolves connection details using flags + config.
func getClient(cCtx *cli.Context) (*client.Client, error) {
	url := cCtx.String("url")
	token := cCtx.String("token")
	alias := cCtx.String("instance")

	cfg := cCtx.App.Metadata["config"].(*config.Config)

	// Always attempt to resolve via config first (required alias). Overrides win.
	cfgURL, cfgToken, err := cfg.Resolve(alias)
	if err != nil {
		return nil, err
	}
	if url == "" {
		url = cfgURL
	}
	if token == "" {
		token = cfgToken
	}

	if url == "" {
		return nil, errors.New("Cockpit base URL not provided (flag --url or config)")
	}
	if token == "" {
		return nil, errors.New("API token not provided (flag --token / COCKPIT_TOKEN or config)")
	}

	return client.New(url, token)
}
