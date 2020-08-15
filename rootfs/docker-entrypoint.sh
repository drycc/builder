#!/usr/bin/env bash

set -e

/bin/create_bucket
exec "$@"
