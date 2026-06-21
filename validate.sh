#!/bin/bash
set -e

# mise
eval "$(mise activate bash)"
mise fmt
mise install

# Run shared lint tasks
mise run gha-lint
mise run shell-lint

# Regenerate OGP cards. Output is deterministic, so the git diff check below
# flags any card that was not regenerated and committed.
go run ./tools/ogp content/articles/*/

# Check for uncommitted changes
git diff --exit-code
