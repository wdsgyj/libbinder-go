#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ADB_BIN="${ADB_BIN:-adb}"
ANDROID_SERIAL="${ANDROID_SERIAL:-}"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-aidl-device-gate.sh

Runs the currently available real-device Binder regression gate:
  1. cmd/cmd callback flow regression
  2. cmd/input regression
  3. cmd/service and cmd/dumpsys protocol regression
EOF
}

main() {
  if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
  fi

  ADB_BIN="${ADB_BIN}" \
  ANDROID_SERIAL="${ANDROID_SERIAL}" \
  "${ROOT_DIR}/scripts/android-device-protocol-regression.sh"
}

main "$@"
