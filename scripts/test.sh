#!/usr/bin/env bash

time go test -timeout=30s -cover -race $(go list ./... | grep -v vendor)
