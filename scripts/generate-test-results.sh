#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

data_dir="${root_dir}/data"
output_file="${data_dir}/test_results.json"

mod_cache="${GOMODCACHE:-${XDG_CACHE_HOME:-$HOME/.cache}/go/mod}"
build_cache="${GOCACHE:-${XDG_CACHE_HOME:-$HOME/.cache}/go/build}"

mkdir -p "${data_dir}" "${mod_cache}" "${build_cache}"

export GOMODCACHE="${mod_cache}"
export GOCACHE="${build_cache}"

go test -json ./... | go run ./cmd/docs-test-results > "${output_file}.tmp"

mv "${output_file}.tmp" "${output_file}"

echo "Wrote ${output_file}"
