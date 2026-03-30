#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ADB_BIN="${ADB_BIN:-adb}"
ANDROID_SERIAL="${ANDROID_SERIAL:-}"
REMOTE_BIN="${REMOTE_BIN:-/data/local/tmp/libbinder-go-cmd}"
TIMEOUT_SEC="${TIMEOUT_SEC:-6}"
TRACE_ENABLED="${TRACE_ENABLED:-0}"
TRACE_FILE="${TRACE_FILE:-/data/local/tmp/libbinder-go-cmd-trace.out}"
TRACE_IPC_FILE="${TRACE_IPC_FILE:-/data/local/tmp/libbinder-go-trace-ipc.bin}"

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-device-cmd-callback-test.sh

Environment:
  ADB_BIN          adb binary path. Default: adb
  ANDROID_SERIAL   target device serial. Auto-detected when exactly one device is online.
  REMOTE_BIN       remote binary path. Default: /data/local/tmp/libbinder-go-cmd
  TIMEOUT_SEC      device-side timeout seconds. Default: 6
  TRACE_ENABLED    set to 1 to enable LIBBINDER_GO_TRACE on device. Default: 0
  TRACE_FILE       remote trace output file. Default: /data/local/tmp/libbinder-go-cmd-trace.out
  TRACE_IPC_FILE   remote dump path used by trace-ipc stop. Default: /data/local/tmp/libbinder-go-trace-ipc.bin
EOF
}

log() {
  echo "[android-device-cmd-callback-test] $*"
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

main() {
  if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
  fi

  command -v "${ADB_BIN}" >/dev/null 2>&1 || die "adb not found: ${ADB_BIN}"
  local serial=""
  serial="$(detect_serial)"

  log "using device ${serial}"
  log "building cmd/cmd for android/arm64"
  (
    cd "${ROOT_DIR}"
    GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -o /tmp/libbinder-go-cmd ./cmd/cmd
  )

  log "pushing binary to ${REMOTE_BIN}"
  "${ADB_BIN}" -s "${serial}" push /tmp/libbinder-go-cmd "${REMOTE_BIN}" >/dev/null
  "${ADB_BIN}" -s "${serial}" shell "chmod 755 $(shell_quote "${REMOTE_BIN}")"

  local trace_prefix=""
  if [ "${TRACE_ENABLED}" = "1" ]; then
    trace_prefix="env LIBBINDER_GO_TRACE=1 "
  fi

  log "case: activity help result receiver"
  adb_capture "${serial}" "timeout ${TIMEOUT_SEC} /system/bin/cmd activity help >/dev/null 2>&1"
  local system_activity_rc="${ADB_CAPTURE_RC}"
  adb_capture "${serial}" "timeout ${TIMEOUT_SEC} ${trace_prefix}$(join_shell_words "${REMOTE_BIN}" activity help) >$(shell_quote "${TRACE_FILE}") 2>&1"
  local go_activity_rc="${ADB_CAPTURE_RC}"
  assert_eq "${go_activity_rc}" "${system_activity_rc}" "activity help exit code"

  log "case: input keyevent result receiver"
  adb_capture "${serial}" "timeout ${TIMEOUT_SEC} /system/bin/cmd input keyevent 0 >/dev/null 2>&1"
  local system_input_rc="${ADB_CAPTURE_RC}"
  adb_capture "${serial}" "timeout ${TIMEOUT_SEC} ${trace_prefix}$(join_shell_words "${REMOTE_BIN}" input keyevent 0) >/dev/null 2>&1"
  local go_input_rc="${ADB_CAPTURE_RC}"
  assert_eq "${go_input_rc}" "${system_input_rc}" "input keyevent exit code"

  log "case: activity trace-ipc shell callback"
  adb_capture "${serial}" "rm -f $(shell_quote "${TRACE_IPC_FILE}") && cmd activity trace-ipc start >/dev/null 2>&1 && timeout ${TIMEOUT_SEC} /system/bin/cmd activity trace-ipc stop --dump-file $(shell_quote "${TRACE_IPC_FILE}") >/dev/null 2>&1; rc=\$?; if [ -s $(shell_quote "${TRACE_IPC_FILE}") ]; then echo RC:\$rc FILE:present; else echo RC:\$rc FILE:missing; fi"
  local system_trace_summary="${ADB_CAPTURE_OUT}"

  adb_capture "${serial}" "rm -f $(shell_quote "${TRACE_IPC_FILE}") $(shell_quote "${TRACE_FILE}") && cmd activity trace-ipc start >/dev/null 2>&1 && timeout ${TIMEOUT_SEC} ${trace_prefix}$(join_shell_words "${REMOTE_BIN}" activity trace-ipc stop --dump-file "${TRACE_IPC_FILE}") >$(shell_quote "${TRACE_FILE}") 2>&1; rc=\$?; if [ -s $(shell_quote "${TRACE_IPC_FILE}") ]; then echo RC:\$rc FILE:present; else echo RC:\$rc FILE:missing; fi"
  local summary="${ADB_CAPTURE_OUT}"
  if [ "${summary}" != "${system_trace_summary}" ]; then
    if [ "${TRACE_ENABLED}" = "1" ]; then
      "${ADB_BIN}" -s "${serial}" shell "tail -n 80 $(shell_quote "${TRACE_FILE}")" >&2 || true
    fi
    die "unexpected trace-ipc summary: got ${summary}, want ${system_trace_summary}"
  fi

  log "pass"
  log "activity help exit code: ${go_activity_rc}"
  log "input keyevent exit code: ${go_input_rc}"
  log "trace-ipc summary: ${summary}"
}

main "$@"
