#!/usr/bin/env bash
go test -timeout 20m -p=1 $(go list ./... | grep 'e2e')
