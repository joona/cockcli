package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	"gopkg.in/yaml.v3"
)

// GetCmd returns `get` command definition supporting JSON (default) or YAML output
func GetCmd() *cli.Command {
	return &cli.Command{
		Name:      "get",
		Usage:     "Download a document and save it under docs/<collection>/ (pretty‑printed JSON or YAML)",
		ArgsUsage: "[--format json|yaml] <collection> <id>",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Value:   "json",
				Usage:   "output format: json or yaml",
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Value:   "",
				Usage:   "output file path (default: docs/<collection>/<id>.json)",
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 2 {
				return cli.Exit("collection and id are required", 1)
			}
			coll := cCtx.Args().Get(0)
			id := cCtx.Args().Get(1)

			format := strings.ToLower(cCtx.String("format"))
			if format != "json" && format != "yaml" {
				return cli.Exit("format must be json or yaml", 1)
			}

			cl, err := getClient(cCtx)
			if err != nil {
				return err
			}

			doc, err := cl.GetDoc(coll, id)
			if err != nil {
				return err
			}

			var outBytes []byte
			var filename string
			switch format {
			case "json":
				// pretty print JSON
				var pretty bytes.Buffer
				if err := json.Indent(&pretty, doc.Raw, "", "  "); err != nil {
					return fmt.Errorf("failed to format JSON: %w", err)
				}
				outBytes = pretty.Bytes()
				if cCtx.String("output") != "" {
					filename = cCtx.String("output")
				} else {
					filename = filepath.Join("docs", coll, fmt.Sprintf("%s.json", id))
				}
			case "yaml":
				var data interface{}
				if err := json.Unmarshal(doc.Raw, &data); err != nil {
					return fmt.Errorf("decode json: %w", err)
				}
				y, err := yaml.Marshal(data)
				if err != nil {
					return fmt.Errorf("encode yaml: %w", err)
				}
				outBytes = y
				if cCtx.String("output") != "" {
					filename = cCtx.String("output")
				} else {
					filename = filepath.Join("docs", coll, fmt.Sprintf("%s.yaml", id))
				}
			}

			// ensure docs/<collection> dir exists
			if err := os.MkdirAll(filepath.Join("docs", coll), 0o755); err != nil {
				return err
			}

			if err := os.WriteFile(filename, outBytes, 0o644); err != nil {
				return err
			}

			fmt.Printf("saved %s (rev %d) -> %s\n", doc.ID, doc.Rev, filename)
			return nil
		},
	}
}
