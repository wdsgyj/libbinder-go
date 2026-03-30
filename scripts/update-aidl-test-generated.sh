#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AIDL_ROOTS=(
  "${ROOT_DIR}/tests/aidl/android/shared/src/main/aidl"
  "${ROOT_DIR}/tests/aidl/extra/aidl"
)
GO_OUT="${ROOT_DIR}/tests/aidl/go/shared/generated"
GO_IMPORT_ROOT="github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated"
TYPES_PATH="${ROOT_DIR}/tests/aidl/extra/aidl/aidl.types.json"

aidl_files=()
for root in "${AIDL_ROOTS[@]}"; do
  if [ ! -d "${root}" ]; then
    continue
  fi
  while IFS= read -r path; do
    aidl_files+=("${path}")
  done < <(find "${root}" -type f -name '*.aidl' | sort)
done

if [ "${#aidl_files[@]}" -eq 0 ]; then
  echo "no AIDL files found under configured roots" >&2
  exit 1
fi

rm -rf "${GO_OUT}"
mkdir -p "${GO_OUT}"

args=(
  ./cmd/aidlgen
  -format go
  -out "${GO_OUT}"
  -go-import-root "${GO_IMPORT_ROOT}"
)

if [ -f "${TYPES_PATH}" ]; then
  args+=(-types "${TYPES_PATH}")
fi

go run "${args[@]}" "${aidl_files[@]}"

echo "generated Go AIDL fixtures into ${GO_OUT}"
