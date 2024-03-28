#!/bin/bash
set -e

exec tini -- harvester-network-migration-helper "${@}"
