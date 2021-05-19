#!/bin/bash

set -e

go build

CONFIG="./configs/config.yaml" \
  LOG_LEVEL="debug" \
  DB_MIGRATION="on" \
  ./trading-service

