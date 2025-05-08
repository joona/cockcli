# Cockcli – Cockpit CMS command‑line interface

Minimal Go CLI that talks to Cockpit CMS JSON API.

```bash
# build
$ go install github.com/joona/cockcli@latest

# config (~/.config/cockcli/config.yaml)
apiKey: global‑token
instances:
  mysite:
    url: https://your-cockpit-instance.com
    apiKey: per‑instance‑token   # optional
```

## Commands

| Command                    | Purpose                                                 |
| -------------------------- | ------------------------------------------------------- |
| `list [collection]`        | list all collections *or* documents of a collection     |
| `get <collection> <id>`    | download a single document to `docs/` for editing       |
| `update <collection> <file>` | upload local file back to CMS with optimistic lock     |

Each command requires `--instance <alias>` (or `-i`) so config can resolve URL & token. Override with `--url` or `--token` when needed.

Examples:

```bash
# list collections on mysite instance
cockcli -i mysite list

# list all docs in `posts` collection
cockcli -i mysite list posts

# download document to docs/posts-<id>.json
cockcli -i mysite get posts 64ce2e63a2ca28d54

# edit with your editor, then upload; fails if server rev changed
cockcli -i mysite update posts docs/posts-64ce2e63a2ca28d54.json
```

