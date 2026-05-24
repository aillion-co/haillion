---
description: Refresh the vendored upstream docs in docs/vendor/.
---

Run `./scripts/gen-docs.sh` (or `make docs`). It reads `docs/manifest.tsv` and writes one snapshot per product into `docs/vendor/`, plus `docs/vendor/INDEX.md`. `docs/vendor/` is generated locally and gitignored — nothing is committed. After running, read `docs/vendor/INDEX.md` and report which products fetched OK and which are MISS. Run this where outbound network is open — in a GitHub-only sandbox the `llms-txt` rows will MISS and must be refreshed locally or in CI.
