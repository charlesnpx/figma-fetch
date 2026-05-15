---
name: figma-fetch
description: "Use this skill for read-only Figma/FigJam source research from Figma URLs."
---

# figma-fetch

Run `figma-fetch <url>` for read-only Figma or FigJam retrieval. Prefer `content.md` for normal context, then `content.json` or `raw/nodes.json` when the task needs more structure.

When available, check setup first:

```bash
mise-en-place setup figma-fetch --capability read
```

Useful forms:

```bash
figma-fetch "https://www.figma.com/design/FILE_KEY/name?node-id=12-34"
figma-fetch "https://www.figma.com/board/FILE_KEY/name?node-id=12-34" --render png
```

The CLI writes a simple output directory with `content.md`, `content.json`, `raw/nodes.json`, and optional rendered assets. The output directory defaults to `~/.cache/figma-fetch/outputs/...`; pass `--out <dir>` only when caller-local files are required. It does not publish context handles or schema manifests.
