#!/bin/bash
set -euo pipefail

# This script builds the application on AWS by:
# 1. Installing all dependencies
# 2. Running go build

go mod download
go build -o ./bin/api-srv
