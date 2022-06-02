#!/usr/bin/env bash
go test -timeout 45m -p=1 $(go list ./... | grep 'e2e')
