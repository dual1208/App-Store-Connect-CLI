#!/bin/sh
set -eu

test_state_dir=$(mktemp -d "${TMPDIR:-/tmp}/asc-file-only-tests.XXXXXX")
trap 'rm -rf "$test_state_dir"' EXIT HUP INT TERM

export ASC_BYPASS_KEYCHAIN=1
export ASC_CONFIG_PATH="$test_state_dir/config/config.json"
export ASC_WEB_SESSION_CACHE_DIR="$test_state_dir/web-sessions"
export ASC_WEB_PASSWORD_STORE_DIR="$test_state_dir/web-passwords"
unset ASC_KEY_ID ASC_ISSUER_ID ASC_PRIVATE_KEY ASC_PRIVATE_KEY_B64 ASC_PRIVATE_KEY_PATH ASC_PROFILE

go test "$@"
