#!/usr/bin/env bash
CGO_ENABLED=0 go build -o ./bin/NDP-scheduler ./src/*.go