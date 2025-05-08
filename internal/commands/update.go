// internal/commands/update.go – YAML→JSON, **JSON‑aware diff**, optimistic‑locking, --dry-run, --format
package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"
	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"gopkg.in/yaml.v3"
)

// UpdateCmd returns the `update` command definition following the common option pattern.
func UpdateCmd() *cli.Command {
	return &cli.Command{
		Name:      "update",
		Usage:     "Upload a local document file (JSON or YAML) with optimistic locking",
		ArgsUsage: "<collection> [id]",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Usage:   "Show JSON‑aware diff and REST call that would be made, but do not persist changes",
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Specify the format of the document file (json or yaml)",
				Value:   "json",
			},
			&cli.StringFlag{
				Name:    "tempfile",
				Aliases: []string{"t"},
				Usage:   "Provide the document contents as a temporary file",
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() < 1 || (cCtx.NArg() < 2 && cCtx.String("tempfile") == "") {
				return cli.Exit("collection and either document ID or --tempfile are required", 1)
			}

			coll := cCtx.Args().Get(0)
			id := cCtx.Args().Get(1) // May be empty if using --tempfile
			format := strings.ToLower(cCtx.String("format"))
			tempfile := cCtx.String("tempfile")

			if format != "json" && format != "yaml" {
				return cli.Exit("invalid format: must be 'json' or 'yaml'", 1)
			}

			var raw []byte
			var err error

			// Read the document file
			if tempfile != "" {
				raw, err = os.ReadFile(tempfile)
				if err != nil {
					return fmt.Errorf("failed to read tempfile: %w", err)
				}
			} else {
				ext := "json"
				if format == "yaml" {
					ext = "yaml"
				}
				filePath := filepath.Join("docs", coll, fmt.Sprintf("%s.%s", id, ext))
				raw, err = os.ReadFile(filePath)
				if err != nil {
					return fmt.Errorf("failed to read file: %w", err)
				}
			}

			// Convert YAML to JSON if needed
			if format == "yaml" {
				var y interface{}
				if err := yaml.Unmarshal(raw, &y); err != nil {
					return fmt.Errorf("invalid YAML: %w", err)
				}
				raw, err = json.MarshalIndent(y, "", "  ")
				if err != nil {
					return err
				}
			}

			// Extract metadata (_id, _modified) from JSON
			var meta struct {
				ID  string `json:"_id"`
				Rev int64  `json:"_modified"`
			}
			if err := json.Unmarshal(raw, &meta); err != nil {
				return fmt.Errorf("invalid JSON: %w", err)
			}

			if meta.ID == "" {
				if id == "" {
					return fmt.Errorf("JSON missing _id field, and no document ID provided")
				}
				meta.ID = id
			}

			// Build Cockpit client
			cl, err := getClient(cCtx)
			if err != nil {
				return err
			}

			// Fetch latest revision to compare
			svrDoc, err := cl.GetDoc(coll, meta.ID)
			if err != nil {
				return err
			}

			// Marshal both docs to canonical JSON for diffing
			svrJSON, _ := json.MarshalIndent(svrDoc.Raw, "", "  ") // Raw should be map[string]any in client doc type
			locJSON := raw                                         // already pretty or compact; we'll diff logically anyway

			// JSON‑aware diff
			differ := diff.New()
			d, err := differ.Compare(svrJSON, locJSON)
			if err != nil {
				return err
			}

			if !d.Modified() {
				fmt.Println("No changes detected – nothing to do.")
				return nil
			}

			// ascii formatter highlights object/array/key changes
			var svrParsed interface{}
			if err := json.Unmarshal(svrJSON, &svrParsed); err != nil {
				return err
			}
			formatterCfg := formatter.AsciiFormatterConfig{
				ShowArrayIndex: true,
				Coloring:       true,
			}
			f := formatter.NewAsciiFormatter(svrParsed, formatterCfg)
			ascii, err := f.Format(d)
			if err != nil {
				return err
			}

			// Only show changed lines
			fmt.Println("=== Diff (server → local) ===")
			for _, line := range strings.Split(ascii, "\n") {
				if strings.Contains(line, "+") || strings.Contains(line, "-") {
					fmt.Println(line)
				}
			}

			// Optimistic‑locking: refuse if revision has changed
			if svrDoc.Rev != meta.Rev {
				return fmt.Errorf("document changed on server (server rev %d ≠ local rev %d)", svrDoc.Rev, meta.Rev)
			}

			// Handle --dry-run
			if cCtx.Bool("dry-run") {
				fmt.Printf("[DRY‑RUN] Would update collection '%s' document '%s' (rev %d) via /api/collections/save/%s\n", coll, meta.ID, meta.Rev, coll)
				return nil
			}

			newRev, err := cl.UpdateDoc(coll, locJSON)
			if err != nil {
				return err
			}
			fmt.Printf("updated %s: rev %d -> %d\n", meta.ID, meta.Rev, newRev)
			return nil
		},
	}
}
