#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ADB_BIN="${ADB_BIN:-adb}"
ANDROID_SERIAL="${ANDROID_SERIAL:-}"
TIMEOUT_SEC="${TIMEOUT_SEC:-6}"
TRACE_ENABLED="${TRACE_ENABLED:-0}"
REAL_INPUT_ARGS="${REAL_INPUT_ARGS:-keyevent 0}"
REMOTE_SERVICE_BIN="${REMOTE_SERVICE_BIN:-/data/local/tmp/libbinder-go-service}"
REMOTE_DUMPSYS_BIN="${REMOTE_DUMPSYS_BIN:-/data/local/tmp/libbinder-go-dumpsys}"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-device-protocol-regression.sh

Environment:
  ADB_BIN            adb binary path. Default: adb
  ANDROID_SERIAL     target device serial. Auto-detected when exactly one device is online.
  TIMEOUT_SEC        device-side timeout seconds. Default: 6
  TRACE_ENABLED      set to 1 to enable LIBBINDER_GO_TRACE for cmd callback regression. Default: 0
  REAL_INPUT_ARGS    forwarded to android-device-input-test.sh. Default: "keyevent 0"
  REMOTE_SERVICE_BIN remote path for cmd/service. Default: /data/local/tmp/libbinder-go-service
  REMOTE_DUMPSYS_BIN remote path for cmd/dumpsys. Default: /data/local/tmp/libbinder-go-dumpsys
EOF
}

log() {
  echo "[android-device-protocol-regression] $*"
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

compare_exact() {
  local serial="$1"
  local system_cmd="$2"
  local ours_cmd="$3"
  local label="$4"

  adb_capture "${serial}" "${system_cmd}"
  local system_rc="${ADB_CAPTURE_RC}"
  local system_out="${ADB_CAPTURE_OUT}"

  adb_capture "${serial}" "${ours_cmd}"
  local ours_rc="${ADB_CAPTURE_RC}"
  local ours_out="${ADB_CAPTURE_OUT}"

  assert_eq "${ours_rc}" "${system_rc}" "${label} exit code"
  assert_eq "${ours_out}" "${system_out}" "${label} output"
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

  log "running cmd callback regression"
  ANDROID_SERIAL="${serial}" \
  ADB_BIN="${ADB_BIN}" \
  TRACE_ENABLED="${TRACE_ENABLED}" \
  "${ROOT_DIR}/scripts/android-device-cmd-callback-test.sh"

  log "running input regression"
  ANDROID_SERIAL="${serial}" \
  ADB_BIN="${ADB_BIN}" \
  REAL_INPUT_ARGS="${REAL_INPUT_ARGS}" \
  "${ROOT_DIR}/scripts/android-device-input-test.sh"

  log "building cmd/service and cmd/dumpsys"
  (
    cd "${ROOT_DIR}"
    GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/libbinder-go-service ./cmd/service
    GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/libbinder-go-dumpsys ./cmd/dumpsys
  )

  log "pushing cmd/service and cmd/dumpsys"
  "${ADB_BIN}" -s "${serial}" push /tmp/libbinder-go-service "${REMOTE_SERVICE_BIN}" >/dev/null
  "${ADB_BIN}" -s "${serial}" push /tmp/libbinder-go-dumpsys "${REMOTE_DUMPSYS_BIN}" >/dev/null
  "${ADB_BIN}" -s "${serial}" shell "chmod 755 $(shell_quote "${REMOTE_SERVICE_BIN}") $(shell_quote "${REMOTE_DUMPSYS_BIN}")"

  log "case: service check activity"
  compare_exact \
    "${serial}" \
    "timeout ${TIMEOUT_SEC} /system/bin/service check activity 2>&1" \
    "timeout ${TIMEOUT_SEC} $(join_shell_words "${REMOTE_SERVICE_BIN}" check activity) 2>&1" \
    "service check activity"

  log "case: service list head"
  compare_exact \
    "${serial}" \
    "timeout ${TIMEOUT_SEC} /system/bin/service list 2>&1 | head -n 5" \
    "timeout ${TIMEOUT_SEC} $(join_shell_words "${REMOTE_SERVICE_BIN}" list) 2>&1 | head -n 5" \
    "service list head"

  log "case: dumpsys -l head"
  compare_exact \
    "${serial}" \
    "timeout ${TIMEOUT_SEC} /system/bin/dumpsys -l 2>&1 | head -n 5" \
    "timeout ${TIMEOUT_SEC} $(join_shell_words "${REMOTE_DUMPSYS_BIN}" -l) 2>&1 | head -n 5" \
    "dumpsys -l head"

  log "case: dumpsys --pid activity"
  compare_exact \
    "${serial}" \
    "timeout ${TIMEOUT_SEC} /system/bin/dumpsys --pid activity 2>&1" \
    "timeout ${TIMEOUT_SEC} $(join_shell_words "${REMOTE_DUMPSYS_BIN}" --pid activity) 2>&1" \
    "dumpsys --pid activity"

  log "pass"
}

main "$@"
