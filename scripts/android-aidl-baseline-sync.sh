#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=./lib/android-emulator-common.sh
source "${ROOT_DIR}/scripts/lib/android-emulator-common.sh"

ANDROID_API_LEVEL="${ANDROID_API_LEVEL:-35}"
ANDROID_IMAGE_TAG="${ANDROID_IMAGE_TAG:-google_apis}"
ANDROID_ABI="${ANDROID_ABI:-arm64-v8a}"
ANDROID_DEVICE_PROFILE="${ANDROID_DEVICE_PROFILE:-medium_phone}"
ANDROID_AVD_NAME="${ANDROID_AVD_NAME:-libbinder-go-api${ANDROID_API_LEVEL}-${ANDROID_ABI}}"
ANDROID_EMULATOR_PORT="${ANDROID_EMULATOR_PORT:-5562}"
ANDROID_SERIAL="${ANDROID_SERIAL:-emulator-${ANDROID_EMULATOR_PORT}}"
ANDROID_FIXTURE_AS_ROOT="${ANDROID_FIXTURE_AS_ROOT:-1}"
ANDROID_HEADLESS="${ANDROID_HEADLESS:-1}"
ANDROID_WIPE_DATA="${ANDROID_WIPE_DATA:-0}"
ANDROID_KEEP_EMULATOR="${ANDROID_KEEP_EMULATOR:-0}"
GRADLE_BIN="${GRADLE_BIN:-gradle}"
REMOTE_DIR="${REMOTE_DIR:-/data/local/tmp/libbinder-go-aidl}"
SERVICE_NAME="${SERVICE_NAME:-libbinder.go.aidltest.baseline}"
GO_ENV_ANDROID=(GOOS=android GOARCH=arm64 CGO_ENABLED=0)
JAVA_SERVER_PACKAGE="com.wdsgyj.libbinder.aidltest.javaserver"
JAVA_SERVER_MAIN="com.wdsgyj.libbinder.aidltest.javaserver.FixtureServerMain"
GO_CLIENT_PKG="./tests/aidl/go/client/baseline"
REMOTE_GO_CLIENT="${REMOTE_DIR}/baseline-go-client"
REMOTE_JAVA_SERVER_LOG="${REMOTE_DIR}/java-server.log"

cleanup() {
  local exit_code=$?

  if [ -n "${JAVA_SERVER_PID:-}" ]; then
    "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "kill ${JAVA_SERVER_PID} >/dev/null 2>&1 || true" >/dev/null 2>&1 || true
  fi

  if [ "${exit_code}" -ne 0 ]; then
    dump_java_server_log || true
  fi

  if [ "${ANDROID_EMULATOR_STARTED:-0}" = "1" ]; then
    android_stop_emulator "${ANDROID_SERIAL}"
  fi

  exit "${exit_code}"
}
trap cleanup EXIT

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-aidl-baseline-sync.sh

This script runs the first interoperability slice:
  Java server -> Go client for IBaselineService

Steps:
  1. prepare emulator or reuse an existing device
  2. build and install the Java fixture server APK
  3. build and push the Go baseline client binary
  4. start the Java fixture server via app_process
  5. invoke the Go client and print its JSON result

Environment:
  ANDROID_SERIAL            adb serial to reuse. Default: emulator-5562
  ANDROID_API_LEVEL         emulator API level. Default: 35
  ANDROID_IMAGE_TAG         emulator image tag. Default: google_apis
  ANDROID_ABI               emulator ABI. Default: arm64-v8a
  ANDROID_FIXTURE_AS_ROOT   set to 1 to launch the Java fixture under su 0 when available
  GRADLE_BIN                Gradle executable. Default: gradle
  REMOTE_DIR                device-side working dir
  SERVICE_NAME              service manager name for the fixture service
EOF
}

log() {
  echo "[android-aidl-baseline-sync] $*"
}

dump_java_server_log() {
  log "device java server log:"
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "cat $(android_shell_quote "${REMOTE_JAVA_SERVER_LOG}") 2>/dev/null || true"
}

build_java_server() {
  log "building Java fixture server APK"
  (
    cd "${ROOT_DIR}/tests/aidl/android"
    "${GRADLE_BIN}" --no-daemon :java-server:assembleDebug
  )
}

find_java_server_apk() {
  find "${ROOT_DIR}/tests/aidl/android/java-server/build/outputs/apk" -type f -name '*debug*.apk' | sort | head -n 1
}

install_java_server() {
  local apk_path="$1"
  [ -n "${apk_path}" ] || android_die "java server APK not found"
  log "installing ${apk_path}"
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" install -r "${apk_path}" >/dev/null
}

build_go_client() {
  log "building Go baseline client"
  (
    cd "${ROOT_DIR}"
    env "${GO_ENV_ANDROID[@]}" go build -o /tmp/baseline-go-client "${GO_CLIENT_PKG}"
  )
}

push_go_client() {
  log "pushing Go baseline client"
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "mkdir -p $(android_shell_quote "${REMOTE_DIR}")" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push /tmp/baseline-go-client "${REMOTE_GO_CLIENT}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "chmod 755 $(android_shell_quote "${REMOTE_GO_CLIENT}")" >/dev/null
}

server_apk_path_on_device() {
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "pm path ${JAVA_SERVER_PACKAGE}" | tr -d '\r' | sed -n 's/^package://p' | head -n 1
}

start_java_server() {
  local apk_path="$1"
  [ -n "${apk_path}" ] || android_die "installed APK path for ${JAVA_SERVER_PACKAGE} not found"

  log "starting Java fixture server via app_process"
  local remote_cmd=""
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "mkdir -p $(android_shell_quote "${REMOTE_DIR}") && : > $(android_shell_quote "${REMOTE_JAVA_SERVER_LOG}")" >/dev/null
  remote_cmd="export CLASSPATH=$(android_shell_quote "${apk_path}"); exec app_process /system/bin ${JAVA_SERVER_MAIN} $(android_shell_quote "${SERVICE_NAME}") >>$(android_shell_quote "${REMOTE_JAVA_SERVER_LOG}") 2>&1 & echo \$!"

  if [ "${ANDROID_FIXTURE_AS_ROOT}" = "1" ] && "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "command -v su >/dev/null 2>&1"; then
    JAVA_SERVER_PID="$("${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "su 0 sh -c $(android_shell_quote "${remote_cmd}")" | tr -d '\r' | tail -n 1)"
  else
    JAVA_SERVER_PID="$("${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "${remote_cmd}" | tr -d '\r' | tail -n 1)"
  fi

  [ -n "${JAVA_SERVER_PID}" ] || android_die "failed to start Java fixture server"
  sleep 2
  log "Java fixture server pid=${JAVA_SERVER_PID}"
}

run_go_client() {
  log "running Go baseline client"
  local remote_cmd=""
  remote_cmd="cd $(android_shell_quote "${REMOTE_DIR}") && exec $(android_join_shell_words "./$(basename "${REMOTE_GO_CLIENT}")" -service "${SERVICE_NAME}")"
  if [ "${ANDROID_FIXTURE_AS_ROOT}" = "1" ] && "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "command -v su >/dev/null 2>&1"; then
    "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "su 0 sh -c $(android_shell_quote "${remote_cmd}")"
    return
  fi
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "${remote_cmd}"
}

prepare_target() {
  local emulator_log=""

  android_setup_paths

  if android_device_online "${ANDROID_SERIAL}"; then
    log "using already connected device ${ANDROID_SERIAL}"
    return 0
  fi

  if [ -d "$(android_avd_dir)" ]; then
    log "using existing AVD ${ANDROID_AVD_NAME}"
  else
    android_ensure_sdk_components
    android_ensure_avd
  fi

  emulator_log="$(mktemp "${TMPDIR:-/tmp}/libbinder-go-aidl-emulator.XXXXXX")"
  android_start_emulator "${ANDROID_SERIAL}" "${ANDROID_EMULATOR_PORT}" "${emulator_log}"
  if [ "${ANDROID_ADB_ROOT:-1}" = "1" ]; then
    android_root_device "${ANDROID_SERIAL}"
  fi
}

main() {
  if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
  fi

  command -v "${GRADLE_BIN}" >/dev/null 2>&1 || android_die "gradle not found: ${GRADLE_BIN}"

  prepare_target
  build_java_server
  install_java_server "$(find_java_server_apk)"
  build_go_client
  push_go_client
  start_java_server "$(server_apk_path_on_device)"
  run_go_client
}

main "$@"
