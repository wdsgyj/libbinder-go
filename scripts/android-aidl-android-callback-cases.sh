#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=./lib/android-emulator-common.sh
source "${ROOT_DIR}/scripts/lib/android-emulator-common.sh"

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
REMOTE_BIN="${REMOTE_BIN:-/data/local/tmp/libbinder-go-cmd}"
TRACE_IPC_FILE="${TRACE_IPC_FILE:-/data/local/tmp/libbinder-go-trace-ipc.bin}"
TIMEOUT_SEC="${TIMEOUT_SEC:-12}"

cleanup() {
  local exit_code=$?

  if [ "${ANDROID_EMULATOR_STARTED:-0}" = "1" ]; then
    android_stop_emulator "${ANDROID_SERIAL}"
  fi

  exit "${exit_code}"
}
trap cleanup EXIT

log() {
  echo "[android-aidl-android-callback-cases] $*"
}

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-aidl-android-callback-cases.sh

Runs Android callback carrier cases on emulator using cmd/cmd:
  1. ResultReceiver via `activity help`
  2. ResultReceiver via `input keyevent 0`
  3. ShellCallback via `activity trace-ipc stop --dump-file ...`
EOF
}

prepare_target() {
  local emulator_log=""

  android_setup_paths

  if android_device_online "${ANDROID_SERIAL}"; then
    log "using already connected device ${ANDROID_SERIAL}"
    return 0
  fi

  if [ ! -d "$(android_avd_dir)" ]; then
    android_ensure_sdk_components
    android_ensure_avd
  fi

  emulator_log="$(mktemp "${TMPDIR:-/tmp}/libbinder-go-aidl-android-callback.XXXXXX")"
  android_start_emulator "${ANDROID_SERIAL}" "${ANDROID_EMULATOR_PORT}" "${emulator_log}"
  android_root_device "${ANDROID_SERIAL}"
}

main() {
  if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
  fi

  prepare_target

  log "running callback carrier regression on ${ANDROID_SERIAL}"
  ANDROID_SERIAL="${ANDROID_SERIAL}" \
  REMOTE_BIN="${REMOTE_BIN}" \
  TRACE_IPC_FILE="${TRACE_IPC_FILE}" \
  TIMEOUT_SEC="${TIMEOUT_SEC}" \
  "${ROOT_DIR}/scripts/android-device-cmd-callback-test.sh"

  log "all Android callback carrier cases passed on ${ANDROID_SERIAL}"
}

main "$@"
