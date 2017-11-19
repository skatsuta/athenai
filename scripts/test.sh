#!/usr/bin/env bash
set -eux

PKG=${1:-./...}
time go test -timeout=30s -cover -race "$PKG"
