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

TEST_REGEX="${TEST_REGEX:-TestRPC(TransactStabilityEnforcement|PrivateVendorRejectsSystemBinder|TCPTransportHelpers|UnixTransportHelpers|WatchDeathOnServerClose)|TestListenRPCUnixAutoAddress|TestRPCTLSTransportHelpers}"

log() {
  echo "[android-aidl-runtime-cases] $*"
}

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-aidl-runtime-cases.sh

Runs runtime Android emulator cases:
  1. stability label enforcement / partition semantics
  2. RPC tcp/unix/tls transport helpers
  3. RPC death notification on server close
EOF
}

main() {
  if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
  fi

  log "running runtime emulator cases on ${ANDROID_SERIAL}"
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
  "${ROOT_DIR}/scripts/android-emulator-test.sh" ./ -- -test.v -test.run "${TEST_REGEX}"

  log "all runtime emulator cases passed on ${ANDROID_SERIAL}"
}

main "$@"
