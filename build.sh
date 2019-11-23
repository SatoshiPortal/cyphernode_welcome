#!/usr/bin/env sh

CGO_ENABLED=0 GOGC=off go build -ldflags "-s" -a
