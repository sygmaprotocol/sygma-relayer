#!/usr/bin/env bash
# The Licensed Work is (c) 2022 Sygma
# SPDX-License-Identifier: BUSL-1.1

go test -timeout 30m -p=1 $(go list ./... | grep 'e2e')
