package commands

import (
	"encoding/json"
	"fmt"

	"github.com/urfave/cli/v2"
)

// ListCmd returns `list` command definition.
func ListCmd() *cli.Command {
	return &cli.Command{
		Name:      "list",
		Usage:     "List documents of a collection or all collection names",
		ArgsUsage: "[collection]",
		Action: func(cCtx *cli.Context) error {
			cl, err := getClient(cCtx)
			if err != nil {
				return err
			}

			if cCtx.NArg() == 0 {
				// List collections
				cols, err := cl.ListCollections()
				if err != nil {
					return err
				}
				for _, n := range cols {
					fmt.Println(n)
				}
				return nil
			}

			coll := cCtx.Args().Get(0)
			docs, err := cl.FetchDocuments(coll)
			if err != nil {
				return err
			}
			for _, raw := range docs {
				var meta struct {
					ID    string `json:"_id"`
					Title string `json:"title,omitempty"`
				}
				_ = json.Unmarshal(raw, &meta)
				label := meta.ID
				if meta.Title != "" {
					label += " - " + meta.Title
				}
				fmt.Println(label)
			}
			return nil
		},
	}
}
