---
name: figma-fetch
description: "Read-only Figma/FigJam retrieval from Figma URLs."
---

# figma-fetch

Use `figma-fetch` when the user needs source context from a Figma or FigJam URL.

```bash
figma-fetch "https://www.figma.com/design/FILE_KEY/name?node-id=12-34"
```

The CLI writes `content.md`, `content.json`, and `raw/nodes.json` to the output directory. Use `--render png` when a selected node image is useful.

Auth is read-only and comes from `FIGMA_TOKEN` unless `--token` is supplied.
