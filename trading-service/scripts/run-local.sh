#!/bin/bash

set -e

go build

CONFIG_LOGGING_LEVEL="debug" \
  CONFIG_DATABASE_MIGRATION="true" \
  ./trading-service

