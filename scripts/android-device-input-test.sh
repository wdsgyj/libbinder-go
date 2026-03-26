#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ADB_BIN="${ADB_BIN:-adb}"
ANDROID_SERIAL="${ANDROID_SERIAL:-}"
REMOTE_BIN="${REMOTE_BIN:-/data/local/tmp/libbinder-go-input}"
REAL_INPUT_ARGS="${REAL_INPUT_ARGS:-keyevent 0}"
TIMEOUT_SEC="${TIMEOUT_SEC:-5}"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-device-input-test.sh

Environment:
  ADB_BIN          adb binary path. Default: adb
  ANDROID_SERIAL   target device serial. Auto-detected when exactly one device is online.
  REMOTE_BIN       remote binary path. Default: /data/local/tmp/libbinder-go-input
  REAL_INPUT_ARGS  real input command args. Default: "keyevent 0"
  TIMEOUT_SEC      device-side timeout seconds. Default: 5

Examples:
  ./scripts/android-device-input-test.sh
  ANDROID_SERIAL=ABC123 ./scripts/android-device-input-test.sh
  REAL_INPUT_ARGS="keyevent 3" ./scripts/android-device-input-test.sh
  REAL_INPUT_ARGS="tap 100 200" ./scripts/android-device-input-test.sh
EOF
}

log() {
  echo "[android-device-input-test] $*"
}

die() {
  echo "error: $*" >&2
  exit 1
}

trim_cr() {
  printf '%s' "$1" | tr -d '\r'
}

shell_quote() {
  printf "'%s'" "$(printf '%s' "$1" | sed "s/'/'\\\\''/g")"
}

join_shell_words() {
  local out=""
  local word=""
  for word in "$@"; do
    if [ -n "${out}" ]; then
      out="${out} "
    fi
    out="${out}$(shell_quote "${word}")"
  done
  printf '%s' "${out}"
}

detect_serial() {
  local devices=()
  local line=""
  while IFS=$'\t' read -r serial state; do
    [ -n "${serial}" ] || continue
    [ "${state}" = "device" ] || continue
    devices+=("${serial}")
  done < <("${ADB_BIN}" devices | tail -n +2)

  if [ -n "${ANDROID_SERIAL}" ]; then
    for line in "${devices[@]}"; do
      if [ "${line}" = "${ANDROID_SERIAL}" ]; then
        printf '%s\n' "${ANDROID_SERIAL}"
        return 0
      fi
    done
    die "device ${ANDROID_SERIAL} is not online"
  fi

  if [ "${#devices[@]}" -eq 1 ]; then
    printf '%s\n' "${devices[0]}"
    return 0
  fi
  if [ "${#devices[@]}" -eq 0 ]; then
    die "no adb device is online"
  fi
  die "multiple adb devices are online; set ANDROID_SERIAL"
}

adb_capture() {
  local serial="$1"
  local cmd="$2"
  local out=""
  set +e
  out="$("${ADB_BIN}" -s "${serial}" shell "${cmd}" 2>&1)"
  ADB_CAPTURE_RC=$?
  set -e
  ADB_CAPTURE_OUT="$(trim_cr "${out}")"
}

first_line() {
  printf '%s\n' "$1" | sed -n '1p'
}

assert_eq() {
  local got="$1"
  local want="$2"
  local what="$3"
  if [ "${got}" != "${want}" ]; then
    echo "expected ${what}: ${want}" >&2
    echo "got ${what}: ${got}" >&2
    exit 1
  fi
}

main() {
  if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
  fi

  command -v "${ADB_BIN}" >/dev/null 2>&1 || die "adb not found: ${ADB_BIN}"
  local serial=""
  serial="$(detect_serial)"

  log "using device ${serial}"
  log "building cmd/input for android/arm64"
  (
    cd "${ROOT_DIR}"
    GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/libbinder-go-input ./cmd/input
  )

  log "pushing binary to ${REMOTE_BIN}"
  "${ADB_BIN}" -s "${serial}" push /tmp/libbinder-go-input "${REMOTE_BIN}" >/dev/null
  "${ADB_BIN}" -s "${serial}" shell "chmod 755 $(shell_quote "${REMOTE_BIN}")"

  log "case: usage output"
  adb_capture "${serial}" "/system/bin/cmd input 2>&1"
  local system_usage_rc="${ADB_CAPTURE_RC}"
  local system_usage_first
  system_usage_first="$(first_line "${ADB_CAPTURE_OUT}")"
  adb_capture "${serial}" "$(shell_quote "${REMOTE_BIN}") 2>&1"
  local ours_usage_rc="${ADB_CAPTURE_RC}"
  local ours_usage_first
  ours_usage_first="$(first_line "${ADB_CAPTURE_OUT}")"
  assert_eq "${ours_usage_rc}" "${system_usage_rc}" "usage exit code"
  assert_eq "${ours_usage_first}" "${system_usage_first}" "usage first line"

  log "case: unknown command"
  adb_capture "${serial}" "/system/bin/cmd input not-a-command 2>&1"
  local system_unknown_rc="${ADB_CAPTURE_RC}"
  local system_unknown_first
  system_unknown_first="$(first_line "${ADB_CAPTURE_OUT}")"
  adb_capture "${serial}" "$(shell_quote "${REMOTE_BIN}") not-a-command 2>&1"
  local ours_unknown_rc="${ADB_CAPTURE_RC}"
  local ours_unknown_first
  ours_unknown_first="$(first_line "${ADB_CAPTURE_OUT}")"
  assert_eq "${ours_unknown_rc}" "${system_unknown_rc}" "unknown exit code"
  assert_eq "${ours_unknown_first}" "${system_unknown_first}" "unknown first line"

  log "case: real input path (${REAL_INPUT_ARGS})"
  local real_args=()
  # shellcheck disable=SC2206
  real_args=(${REAL_INPUT_ARGS})
  local real_cmd=""
  real_cmd="$(join_shell_words /system/bin/cmd input "${real_args[@]}")"
  adb_capture "${serial}" "timeout ${TIMEOUT_SEC} ${real_cmd} >/dev/null 2>&1"
  local system_real_rc="${ADB_CAPTURE_RC}"
  real_cmd="$(join_shell_words "${REMOTE_BIN}" "${real_args[@]}")"
  adb_capture "${serial}" "timeout ${TIMEOUT_SEC} ${real_cmd} >/dev/null 2>&1"
  local ours_real_rc="${ADB_CAPTURE_RC}"
  assert_eq "${ours_real_rc}" "${system_real_rc}" "real input exit code"

  log "pass"
  log "usage first line: ${ours_usage_first}"
  log "unknown first line: ${ours_unknown_first}"
  log "real input args: ${REAL_INPUT_ARGS}"
  log "real input exit code: ${ours_real_rc}"
}

main "$@"
