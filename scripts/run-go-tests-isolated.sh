#!/bin/sh
set -eu

test_state_dir=$(mktemp -d "${TMPDIR:-/tmp}/asc-file-only-tests.XXXXXX")
trap 'rm -rf "$test_state_dir"' EXIT HUP INT TERM

export ASC_CONFIG_PATH="$test_state_dir/config/config.json"
unset ASC_KEY_ID ASC_ISSUER_ID ASC_PRIVATE_KEY ASC_PRIVATE_KEY_B64 ASC_PRIVATE_KEY_PATH ASC_PROFILE

go test "$@"
