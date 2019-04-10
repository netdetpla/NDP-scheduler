#!/usr/bin/env bash
docker run -v /root/ndp/NDP-scheduler/bin/:/ns/bin/ -e "CGO_ENABLED=0" ndp-scheduler-builder:test
