#!/bin/bash

set -e

go build

CONFIG="./configs/config.yaml" \
LOG_LEVEL="debug" \
./trading-service

