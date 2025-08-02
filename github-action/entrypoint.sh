#!/bin/bash
set -e
DIR="$1"
if [ -z "$DIR" ]; then
  # Use GITHUB_WORKSPACE if available, else fallback to git root
  DIR="${GITHUB_WORKSPACE:-$(git rev-parse --show-toplevel)}"
fi
./terralink check --dir "$DIR"

