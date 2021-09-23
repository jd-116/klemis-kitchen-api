#!/bin/bash
set -euo pipefail

# This script copies all files in:
# - aws/bundle-include/*
# - api/*
# into a new zip folder

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

WORK_DIR=$(mktemp -d)
if [[ ! "$WORK_DIR" || ! -d "$WORK_DIR" ]]; then
  echo "Could not create temp working directory"
  exit 1
fi

function cleanup {
  rm -rf "$WORK_DIR"
  echo "Deleted temp working directory $WORK_DIR"
}

trap cleanup EXIT

echo "Using '$WORK_DIR' as a temp working directory"
cp -r "$DIR/../api/"* "$WORK_DIR"
cp -r "$DIR/../aws/bundle-include/"* "$WORK_DIR"
# Delete the binary file if it existed in the api/ folder
rm -f "$WORK_DIR/klemis-kitchen-api"
rm -f "$DIR/../aws-bundle.zip"
pushd "$WORK_DIR"
zip -r "$DIR/../aws-bundle.zip" .
popd
echo "Generated bundle at ./aws-bundle.zip"
