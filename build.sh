#!/usr/bin/env bash
CGO_ENABLED=0 go build -o /ns/bin/NDP-scheduler /ns/src/*.go
