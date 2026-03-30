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
ANDROID_FIXTURE_AS_ROOT="${ANDROID_FIXTURE_AS_ROOT:-1}"
ANDROID_HEADLESS="${ANDROID_HEADLESS:-1}"
ANDROID_WIPE_DATA="${ANDROID_WIPE_DATA:-0}"
ANDROID_KEEP_EMULATOR="${ANDROID_KEEP_EMULATOR:-0}"
GRADLE_BIN="${GRADLE_BIN:-${ROOT_DIR}/tests/aidl/android/gradlew}"
REMOTE_DIR="${REMOTE_DIR:-/data/local/tmp/libbinder-go-aidl}"
GO_ENV_ANDROID=(GOOS=android GOARCH=arm64 CGO_ENABLED=0)

JAVA_SERVER_PACKAGE="com.wdsgyj.libbinder.aidltest.javaserver"
JAVA_CLIENT_PACKAGE="com.wdsgyj.libbinder.aidltest.javaclient"
JAVA_BASELINE_SERVER_MAIN="com.wdsgyj.libbinder.aidltest.javaserver.FixtureServerMain"
JAVA_LIFECYCLE_CLIENT_MAIN="com.wdsgyj.libbinder.aidltest.javaclient.LifecycleClientMain"

BASELINE_SERVICE_NAME="${BASELINE_SERVICE_NAME:-libbinder.go.aidltest.baseline}"

REMOTE_BASELINE_GO_SERVER="${REMOTE_DIR}/baseline-go-server"
REMOTE_LIFECYCLE_GO_CLIENT="${REMOTE_DIR}/lifecycle-go-client"

REMOTE_BASELINE_JAVA_SERVER_LOG="${REMOTE_DIR}/lifecycle-baseline-java-server.log"
REMOTE_BASELINE_GO_SERVER_LOG="${REMOTE_DIR}/lifecycle-baseline-go-server.log"
REMOTE_LIFECYCLE_JAVA_CLIENT_LOG="${REMOTE_DIR}/lifecycle-java-client.log"
REMOTE_LIFECYCLE_GO_CLIENT_LOG="${REMOTE_DIR}/lifecycle-go-client.log"

SERVER_PIDS=()
LAST_BACKGROUND_PID=""

cleanup() {
  local exit_code=$?
  local pid=""

  for pid in "${SERVER_PIDS[@]:-}"; do
    "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "kill ${pid} >/dev/null 2>&1 || true" >/dev/null 2>&1 || true
  done

  if [ "${exit_code}" -ne 0 ]; then
    dump_case_logs || true
  fi

  if [ "${ANDROID_EMULATOR_STARTED:-0}" = "1" ]; then
    android_stop_emulator "${ANDROID_SERIAL}"
  fi

  exit "${exit_code}"
}
trap cleanup EXIT

log() {
  echo "[android-aidl-lifecycle-cases] $*"
}

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-aidl-lifecycle-cases.sh

Runs lifecycle Android emulator AIDL cases:
  1. Java server -> Go client discovery
  2. Go server -> Java client discovery
  3. Java server -> Go client death recipient
  4. Go server -> Java client death recipient
EOF
}

dump_case_logs() {
  local path=""
  log "device fixture logs:"
  for path in \
    "${REMOTE_BASELINE_JAVA_SERVER_LOG}" \
    "${REMOTE_BASELINE_GO_SERVER_LOG}" \
    "${REMOTE_LIFECYCLE_JAVA_CLIENT_LOG}" \
    "${REMOTE_LIFECYCLE_GO_CLIENT_LOG}"; do
    echo "--- ${path} ---"
    "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "cat $(android_shell_quote "${path}") 2>/dev/null || true"
  done
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

  emulator_log="$(mktemp "${TMPDIR:-/tmp}/libbinder-go-aidl-lifecycle.XXXXXX")"
  android_start_emulator "${ANDROID_SERIAL}" "${ANDROID_EMULATOR_PORT}" "${emulator_log}"
  android_root_device "${ANDROID_SERIAL}"
}

build_java_fixtures() {
  log "building Java fixture APKs"
  (
    cd "${ROOT_DIR}/tests/aidl/android"
    "${GRADLE_BIN}" --no-daemon :shared:assembleDebug :java-server:assembleDebug :java-client:assembleDebug
  )
}

find_apk() {
  find "$1" -type f -name '*debug*.apk' | sort | head -n 1
}

install_java_fixtures() {
  local server_apk=""
  local client_apk=""

  server_apk="$(find_apk "${ROOT_DIR}/tests/aidl/android/java-server/build/outputs/apk")"
  client_apk="$(find_apk "${ROOT_DIR}/tests/aidl/android/java-client/build/outputs/apk")"

  [ -n "${server_apk}" ] || android_die "java-server APK not found"
  [ -n "${client_apk}" ] || android_die "java-client APK not found"

  log "installing Java fixture APKs"
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" install -r "${server_apk}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" install -r "${client_apk}" >/dev/null
}

build_go_fixtures() {
  log "building Go fixture binaries"
  (
    cd "${ROOT_DIR}"
    env "${GO_ENV_ANDROID[@]}" go build -o /tmp/baseline-go-server ./tests/aidl/go/server/baseline
    env "${GO_ENV_ANDROID[@]}" go build -o /tmp/lifecycle-go-client ./tests/aidl/go/client/lifecycle
  )
}

push_go_fixtures() {
  log "pushing Go fixture binaries"
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "mkdir -p $(android_shell_quote "${REMOTE_DIR}")" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push /tmp/baseline-go-server "${REMOTE_BASELINE_GO_SERVER}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push /tmp/lifecycle-go-client "${REMOTE_LIFECYCLE_GO_CLIENT}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "chmod 755 $(android_shell_quote "${REMOTE_BASELINE_GO_SERVER}") $(android_shell_quote "${REMOTE_LIFECYCLE_GO_CLIENT}")" >/dev/null
}

installed_apk_path() {
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "pm path $1" | tr -d '\r' | sed -n 's/^package://p' | head -n 1
}

truncate_log() {
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell ": > $(android_shell_quote "$1")" >/dev/null
}

run_root_shell() {
  local cmd="$1"
  if [ "${ANDROID_FIXTURE_AS_ROOT}" = "1" ] && "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "command -v su >/dev/null 2>&1"; then
    "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "su 0 sh -c $(android_shell_quote "${cmd}")"
    return
  fi
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "${cmd}"
}

start_background() {
  local log_path="$1"
  shift
  local cmd="$*"
  local pid=""

  truncate_log "${log_path}"
  pid="$(run_root_shell "${cmd} >>$(android_shell_quote "${log_path}") 2>&1 & echo \$!" | tr -d '\r' | tail -n 1)"
  [ -n "${pid}" ] || android_die "failed to start background command: ${cmd}"
  LAST_BACKGROUND_PID="${pid}"
  SERVER_PIDS+=("${pid}")
  sleep 1
}

stop_last_server() {
  local count="${#SERVER_PIDS[@]}"
  local idx=""
  local pid=""
  if [ "${count}" -eq 0 ]; then
    return 0
  fi
  idx=$((count - 1))
  pid="${SERVER_PIDS[${idx}]}"
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "kill ${pid} >/dev/null 2>&1 || true" >/dev/null 2>&1 || true
  unset 'SERVER_PIDS[idx]'
}

run_foreground_to_log() {
  local log_path="$1"
  shift
  local cmd="$*"

  truncate_log "${log_path}"
  run_root_shell "${cmd} >$(android_shell_quote "${log_path}") 2>&1"
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "cat $(android_shell_quote "${log_path}")"
}

start_java_server() {
  local service_name="$1"
  local log_path="$2"
  local apk_path=""

  apk_path="$(installed_apk_path "${JAVA_SERVER_PACKAGE}")"
  [ -n "${apk_path}" ] || android_die "installed APK path for ${JAVA_SERVER_PACKAGE} not found"
  start_background "${log_path}" "export CLASSPATH=$(android_shell_quote "${apk_path}"); exec app_process /system/bin ${JAVA_BASELINE_SERVER_MAIN} $(android_join_shell_words "${service_name}")"
}

start_go_server() {
  local service_name="$1"
  local log_path="$2"
  start_background "${log_path}" "cd $(android_shell_quote "${REMOTE_DIR}") && exec ./$(basename "${REMOTE_BASELINE_GO_SERVER}") $(android_join_shell_words "-service" "${service_name}")"
}

run_java_client() {
  local mode="$1"
  local service_name="$2"
  local expected_prefix="$3"
  local kill_pid="$4"
  local log_path="$5"
  local apk_path=""

  apk_path="$(installed_apk_path "${JAVA_CLIENT_PACKAGE}")"
  [ -n "${apk_path}" ] || android_die "installed APK path for ${JAVA_CLIENT_PACKAGE} not found"
  run_foreground_to_log "${log_path}" "export CLASSPATH=$(android_shell_quote "${apk_path}"); exec app_process /system/bin ${JAVA_LIFECYCLE_CLIENT_MAIN} $(android_join_shell_words "${mode}" "${service_name}" "${expected_prefix}" "${kill_pid}" "500")"
}

run_go_client() {
  local mode="$1"
  local service_name="$2"
  local expected_prefix="$3"
  local kill_pid="$4"
  local log_path="$5"

  run_foreground_to_log "${log_path}" "cd $(android_shell_quote "${REMOTE_DIR}") && exec ./$(basename "${REMOTE_LIFECYCLE_GO_CLIENT}") $(android_join_shell_words "-mode" "${mode}" "-service" "${service_name}" "-expect-prefix" "${expected_prefix}" "-kill-pid" "${kill_pid}" "-kill-delay" "500ms")"
}

run_case_java_server_go_client_discovery() {
  log "case: java_server_go_client lifecycle discovery"
  start_java_server "${BASELINE_SERVICE_NAME}" "${REMOTE_BASELINE_JAVA_SERVER_LOG}"
  run_go_client "discovery" "${BASELINE_SERVICE_NAME}" "java" "0" "${REMOTE_LIFECYCLE_GO_CLIENT_LOG}"
  stop_last_server
}

run_case_go_server_java_client_discovery() {
  log "case: go_server_java_client lifecycle discovery"
  start_go_server "${BASELINE_SERVICE_NAME}" "${REMOTE_BASELINE_GO_SERVER_LOG}"
  run_java_client "discovery" "${BASELINE_SERVICE_NAME}" "go" "0" "${REMOTE_LIFECYCLE_JAVA_CLIENT_LOG}"
  stop_last_server
}

run_case_java_server_go_client_death() {
  local pid=""
  log "case: java_server_go_client death"
  start_java_server "${BASELINE_SERVICE_NAME}" "${REMOTE_BASELINE_JAVA_SERVER_LOG}"
  pid="${LAST_BACKGROUND_PID}"
  run_go_client "death" "${BASELINE_SERVICE_NAME}" "java" "${pid}" "${REMOTE_LIFECYCLE_GO_CLIENT_LOG}"
  stop_last_server
}

run_case_go_server_java_client_death() {
  local pid=""
  log "case: go_server_java_client death"
  start_go_server "${BASELINE_SERVICE_NAME}" "${REMOTE_BASELINE_GO_SERVER_LOG}"
  pid="${LAST_BACKGROUND_PID}"
  run_java_client "death" "${BASELINE_SERVICE_NAME}" "go" "${pid}" "${REMOTE_LIFECYCLE_JAVA_CLIENT_LOG}"
  stop_last_server
}

main() {
  if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
  fi

  if [[ "${GRADLE_BIN}" == */* ]]; then
    [ -x "${GRADLE_BIN}" ] || android_die "gradle wrapper not found: ${GRADLE_BIN}"
  else
    command -v "${GRADLE_BIN}" >/dev/null 2>&1 || android_die "gradle not found: ${GRADLE_BIN}"
  fi

  prepare_target
  build_java_fixtures
  install_java_fixtures
  build_go_fixtures
  push_go_fixtures

  run_case_java_server_go_client_discovery
  run_case_go_server_java_client_discovery
  run_case_java_server_go_client_death
  run_case_go_server_java_client_death

  log "all lifecycle AIDL cases passed on ${ANDROID_SERIAL}"
}

main "$@"
