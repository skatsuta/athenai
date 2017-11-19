#!/usr/bin/env bash
set -eux

time go test -timeout=30s -cover -race $(go list ./... | grep -v vendor)
