# figma-fetch

`figma-fetch` is a small read-only CLI for turning a Figma or FigJam URL into files an agent can read without keeping the API response in the conversation.

It writes:

- `content.md`
- `content.json`
- `raw/nodes.json`
- `assets/<node>.<fmt>` when `--render` is used

## Install

```bash
go install github.com/charlesnpx/figma-fetch/cmd/figma-fetch@latest
```

For delegated installers:

```bash
./install-skill.sh --plan --target all --json
./install-skill.sh --install --target all --json
```

## Usage

```bash
export FIGMA_TOKEN=...
mise-en-place setup figma-fetch --capability read
figma-fetch "https://www.figma.com/design/FILE_KEY/name?node-id=12-34"
figma-fetch "https://www.figma.com/board/FILE_KEY/name?node-id=12-34" --render png
```

Flags:

- `--node <id>` overrides the URL node id.
- `--out <dir>` defaults to `~/.cache/figma-fetch/outputs/<file_key>/<node_or_root>`.
- `--cache-dir <dir>` defaults to `~/.cache/figma-fetch`.
- `--no-cache` bypasses cache reads and writes.
- `--render <fmt>` renders the selected node as `png`, `svg`, `pdf`, or `jpg`.
- `--token <pat>` defaults to `$FIGMA_TOKEN`.

Pass an explicit relative `--out` if you want output materialized in the caller's directory.

The delegated installer declares `FIGMA_TOKEN` as a secret read setup requirement for `mise-en-place setup`.

The cache is intentionally simple: files are keyed by a stable SHA-256 hash of fetch parameters. There is no expiry policy. Remove `~/.cache/figma-fetch` to clear it.
