#!/usr/bin/env bash
docker run -v /root/ndp/NDP-scheduler/bin/:/ns/bin/ -v /root/ndp/NDP-scheduler/src/:/ns/src/ -e "CGO_ENABLED=0" ndp-scheduler-builder
