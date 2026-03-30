#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

ANDROID_API_LEVEL="${ANDROID_API_LEVEL:-35}"
ANDROID_IMAGE_TAG="${ANDROID_IMAGE_TAG:-default}"
ANDROID_ABI="${ANDROID_ABI:-arm64-v8a}"
ANDROID_DEVICE_PROFILE="${ANDROID_DEVICE_PROFILE:-medium_phone}"
ANDROID_AVD_NAME="${ANDROID_AVD_NAME:-libbinder-go-api${ANDROID_API_LEVEL}-${ANDROID_ABI}}"
ANDROID_EMULATOR_PORT="${ANDROID_EMULATOR_PORT:-5562}"
ANDROID_SERIAL="${ANDROID_SERIAL:-emulator-${ANDROID_EMULATOR_PORT}}"
ANDROID_HEADLESS="${ANDROID_HEADLESS:-1}"
ANDROID_WIPE_DATA="${ANDROID_WIPE_DATA:-0}"
ANDROID_KEEP_EMULATOR="${ANDROID_KEEP_EMULATOR:-0}"

log() {
  echo "[android-aidl-full-emulator] $*"
}

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-aidl-full-emulator.sh

Runs the complete Android emulator AIDL regression gate:
  1. basic slice
  2. advanced slice
  3. extended raw-Map/custom-parcelable slice
  4. governance slice
  5. lifecycle slice
  6. Android callback carrier slice
  7. scale slice
  8. runtime slice
EOF
}

run_step() {
  local script="$1"
  log "running $(basename "${script}")"
  ANDROID_API_LEVEL="${ANDROID_API_LEVEL}" \
  ANDROID_IMAGE_TAG="${ANDROID_IMAGE_TAG}" \
  ANDROID_ABI="${ANDROID_ABI}" \
  ANDROID_DEVICE_PROFILE="${ANDROID_DEVICE_PROFILE}" \
  ANDROID_AVD_NAME="${ANDROID_AVD_NAME}" \
  ANDROID_EMULATOR_PORT="${ANDROID_EMULATOR_PORT}" \
  ANDROID_SERIAL="${ANDROID_SERIAL}" \
  ANDROID_HEADLESS="${ANDROID_HEADLESS}" \
  ANDROID_WIPE_DATA="${ANDROID_WIPE_DATA}" \
  ANDROID_KEEP_EMULATOR="${ANDROID_KEEP_EMULATOR}" \
  "${script}"
}

main() {
  if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
  fi

  run_step "${ROOT_DIR}/scripts/android-aidl-basic-cases.sh"
  run_step "${ROOT_DIR}/scripts/android-aidl-advanced-cases.sh"
  run_step "${ROOT_DIR}/scripts/android-aidl-extended-cases.sh"
  run_step "${ROOT_DIR}/scripts/android-aidl-governance-cases.sh"
  run_step "${ROOT_DIR}/scripts/android-aidl-lifecycle-cases.sh"
  run_step "${ROOT_DIR}/scripts/android-aidl-android-callback-cases.sh"
  run_step "${ROOT_DIR}/scripts/android-aidl-scale-cases.sh"
  run_step "${ROOT_DIR}/scripts/android-aidl-runtime-cases.sh"

  log "full emulator gate passed on ${ANDROID_SERIAL}"
}

main "$@"
