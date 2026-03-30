#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
AIDL_ROOT="${ROOT_DIR}/tests/aidl/android/shared/src/main/aidl"
GO_OUT="${ROOT_DIR}/tests/aidl/go/shared/generated"
GO_IMPORT_ROOT="github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated"

aidl_files=()
while IFS= read -r path; do
  aidl_files+=("${path}")
done < <(find "${AIDL_ROOT}" -type f -name '*.aidl' | sort)

if [ "${#aidl_files[@]}" -eq 0 ]; then
  echo "no AIDL files found under ${AIDL_ROOT}" >&2
  exit 1
fi

rm -rf "${GO_OUT}"
mkdir -p "${GO_OUT}"

go run ./cmd/aidlgen \
  -format go \
  -out "${GO_OUT}" \
  -go-import-root "${GO_IMPORT_ROOT}" \
  "${aidl_files[@]}"

echo "generated Go AIDL fixtures into ${GO_OUT}"
