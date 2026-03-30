#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CATALOG_MD="${ROOT_DIR}/tests/aidl/cases/catalog.md"
CATALOG_JSON="${ROOT_DIR}/tests/aidl/cases/catalog.json"
ORDER_MD="${ROOT_DIR}/tests/aidl/cases/implementation-order.md"
FRAMEWORK_MD="${ROOT_DIR}/doc/aidl-full-compat-test-framework.md"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-aidl-matrix-test.sh <command>

Commands:
  layout        Show framework layout files
  phases        Show phased implementation plan
  list-cases    Show the human-readable case catalog
  list-phase N  Show cases in a specific phase from the JSON catalog
  show-case ID  Show a single case record from the JSON catalog
  generate-go   Regenerate Go bindings from shared AIDL fixtures
  baseline-sync Run the first Java server -> Go client baseline script
  basic-emulator Run the current basic emulator matrix
  advanced-emulator Run the current advanced emulator slice
  catalog-json  Print the machine-readable case catalog
  help          Show this help

This script is the entry point for the AIDL compatibility matrix framework.
The executable per-case runner will be added on top of this scaffold.
EOF
}

cmd="${1:-help}"

case "${cmd}" in
  layout)
    cat <<EOF
Framework doc: ${FRAMEWORK_MD}
Catalog doc:   ${CATALOG_MD}
Order doc:     ${ORDER_MD}
Catalog json:  ${CATALOG_JSON}

Directories:
  tests/aidl/host
  tests/aidl/android
  tests/aidl/go
  tests/aidl/cases
EOF
    ;;
  phases)
    cat "${ORDER_MD}"
    ;;
  list-cases)
    cat "${CATALOG_MD}"
    ;;
  list-phase)
    phase="${2:-}"
    if [ -z "${phase}" ]; then
      echo "missing phase number" >&2
      exit 1
    fi
    python3 - "${CATALOG_JSON}" "${phase}" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)

phase = int(sys.argv[2])
for item in data["cases"]:
    if item["phase"] == phase:
        print(f'{item["id"]}\t{item["priority"]}\t{item["area"]}\t{item["title"]}')
PY
    ;;
  show-case)
    case_id="${2:-}"
    if [ -z "${case_id}" ]; then
      echo "missing case id" >&2
      exit 1
    fi
    python3 - "${CATALOG_JSON}" "${case_id}" <<'PY'
import json
import sys

with open(sys.argv[1], "r", encoding="utf-8") as fh:
    data = json.load(fh)

case_id = sys.argv[2]
for item in data["cases"]:
    if item["id"] == case_id:
        print(json.dumps(item, ensure_ascii=False, indent=2))
        break
else:
    raise SystemExit(f"case not found: {case_id}")
PY
    ;;
  generate-go)
    "${ROOT_DIR}/scripts/update-aidl-test-generated.sh"
    ;;
  baseline-sync)
    "${ROOT_DIR}/scripts/android-aidl-baseline-sync.sh" "${@:2}"
    ;;
  basic-emulator)
    "${ROOT_DIR}/scripts/android-aidl-basic-cases.sh" "${@:2}"
    ;;
  advanced-emulator)
    "${ROOT_DIR}/scripts/android-aidl-advanced-cases.sh" "${@:2}"
    ;;
  catalog-json)
    cat "${CATALOG_JSON}"
    ;;
  help|--help|-h)
    usage
    ;;
  *)
    echo "unknown command: ${cmd}" >&2
    usage >&2
    exit 1
    ;;
esac
