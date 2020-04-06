#!/bin/bash
set -ex

export NOMAD_ADDR=http://$HOSTNAME:4646

exec "$@"
