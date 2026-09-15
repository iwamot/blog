#!/bin/bash
set -e

# mise
eval "$(mise activate bash)"
mise fmt
mise install

# Go
# Licenses that may ship alongside MIT code: permissive ones, plus MPL-2.0,
# whose terms stay with its own files. Anything else fails for a human to read.
allowed_licenses=(
  0BSD
  Apache-2.0
  BlueOak-1.0.0
  BSD-2-Clause
  BSD-3-Clause
  CC0-1.0
  CNRI-Python
  ISC
  MIT
  MIT-0
  MIT-CMU
  MPL-2.0
  PSF-2.0
  Python-2.0
  Zlib
)
go-licenses check ./... --allowed_licenses="$(IFS=',' && echo "${allowed_licenses[*]}")"
govulncheck ./...
go fix ./...
gofmt -w .
go vet ./...

# Run shared lint tasks
mise run gha-lint
mise run shell-lint

# Regenerate OGP cards. Output is deterministic, so the git diff check below
# flags any card that was not regenerated and committed.
go run ./tools/ogp content/articles/*/

# Check for uncommitted changes
git diff --exit-code
