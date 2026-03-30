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
GRADLE_BIN="${GRADLE_BIN:-gradle}"
REMOTE_DIR="${REMOTE_DIR:-/data/local/tmp/libbinder-go-aidl}"
GO_ENV_ANDROID=(GOOS=android GOARCH=arm64 CGO_ENABLED=0)

JAVA_SERVER_PACKAGE="com.wdsgyj.libbinder.aidltest.javaserver"
JAVA_CLIENT_PACKAGE="com.wdsgyj.libbinder.aidltest.javaclient"
JAVA_BASELINE_SERVER_MAIN="com.wdsgyj.libbinder.aidltest.javaserver.FixtureServerMain"
JAVA_MATRIX_SERVER_MAIN="com.wdsgyj.libbinder.aidltest.javaserver.BasicMatrixServerMain"
JAVA_BASELINE_CLIENT_MAIN="com.wdsgyj.libbinder.aidltest.javaclient.FixtureClientMain"
JAVA_MATRIX_CLIENT_MAIN="com.wdsgyj.libbinder.aidltest.javaclient.BasicMatrixClientMain"

BASELINE_SERVICE_NAME="${BASELINE_SERVICE_NAME:-libbinder.go.aidltest.baseline}"
MATRIX_SERVICE_NAME="${MATRIX_SERVICE_NAME:-libbinder.go.aidltest.matrix}"

REMOTE_BASELINE_GO_CLIENT="${REMOTE_DIR}/baseline-go-client"
REMOTE_BASELINE_GO_SERVER="${REMOTE_DIR}/baseline-go-server"
REMOTE_MATRIX_GO_CLIENT="${REMOTE_DIR}/matrix-go-client"
REMOTE_MATRIX_GO_SERVER="${REMOTE_DIR}/matrix-go-server"

REMOTE_BASELINE_JAVA_SERVER_LOG="${REMOTE_DIR}/baseline-java-server.log"
REMOTE_BASELINE_JAVA_CLIENT_LOG="${REMOTE_DIR}/baseline-java-client.log"
REMOTE_BASELINE_GO_SERVER_LOG="${REMOTE_DIR}/baseline-go-server.log"
REMOTE_BASELINE_GO_CLIENT_LOG="${REMOTE_DIR}/baseline-go-client.log"
REMOTE_MATRIX_JAVA_SERVER_LOG="${REMOTE_DIR}/matrix-java-server.log"
REMOTE_MATRIX_JAVA_CLIENT_LOG="${REMOTE_DIR}/matrix-java-client.log"
REMOTE_MATRIX_GO_SERVER_LOG="${REMOTE_DIR}/matrix-go-server.log"
REMOTE_MATRIX_GO_CLIENT_LOG="${REMOTE_DIR}/matrix-go-client.log"

SERVER_PIDS=()

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
  echo "[android-aidl-basic-cases] $*"
}

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-aidl-basic-cases.sh

Runs the basic Android emulator AIDL compatibility slice:
  1. Java server -> Go client (baseline)
  2. Java server -> Go client (matrix)
  3. Go server -> Java client (baseline)
  4. Go server -> Java client (matrix)
EOF
}

dump_case_logs() {
  local path=""
  log "device fixture logs:"
  for path in \
    "${REMOTE_BASELINE_JAVA_SERVER_LOG}" \
    "${REMOTE_BASELINE_JAVA_CLIENT_LOG}" \
    "${REMOTE_BASELINE_GO_SERVER_LOG}" \
    "${REMOTE_BASELINE_GO_CLIENT_LOG}" \
    "${REMOTE_MATRIX_JAVA_SERVER_LOG}" \
    "${REMOTE_MATRIX_JAVA_CLIENT_LOG}" \
    "${REMOTE_MATRIX_GO_SERVER_LOG}" \
    "${REMOTE_MATRIX_GO_CLIENT_LOG}"; do
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

  emulator_log="$(mktemp "${TMPDIR:-/tmp}/libbinder-go-aidl-basic.XXXXXX")"
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
    env "${GO_ENV_ANDROID[@]}" go build -o /tmp/baseline-go-client ./tests/aidl/go/client/baseline
    env "${GO_ENV_ANDROID[@]}" go build -o /tmp/baseline-go-server ./tests/aidl/go/server/baseline
    env "${GO_ENV_ANDROID[@]}" go build -o /tmp/matrix-go-client ./tests/aidl/go/client/matrix
    env "${GO_ENV_ANDROID[@]}" go build -o /tmp/matrix-go-server ./tests/aidl/go/server/matrix
  )
}

push_go_fixtures() {
  log "pushing Go fixture binaries"
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "mkdir -p $(android_shell_quote "${REMOTE_DIR}")" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push /tmp/baseline-go-client "${REMOTE_BASELINE_GO_CLIENT}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push /tmp/baseline-go-server "${REMOTE_BASELINE_GO_SERVER}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push /tmp/matrix-go-client "${REMOTE_MATRIX_GO_CLIENT}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push /tmp/matrix-go-server "${REMOTE_MATRIX_GO_SERVER}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "chmod 755 $(android_shell_quote "${REMOTE_BASELINE_GO_CLIENT}") $(android_shell_quote "${REMOTE_BASELINE_GO_SERVER}") $(android_shell_quote "${REMOTE_MATRIX_GO_CLIENT}") $(android_shell_quote "${REMOTE_MATRIX_GO_SERVER}")" >/dev/null
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
  local main_class="$1"
  local service_name="$2"
  local prefix="$3"
  local log_path="$4"
  local apk_path=""

  apk_path="$(installed_apk_path "${JAVA_SERVER_PACKAGE}")"
  [ -n "${apk_path}" ] || android_die "installed APK path for ${JAVA_SERVER_PACKAGE} not found"
  start_background "${log_path}" "export CLASSPATH=$(android_shell_quote "${apk_path}"); exec app_process /system/bin ${main_class} $(android_join_shell_words "${service_name}" "${prefix}")"
}

run_java_client() {
  local main_class="$1"
  local service_name="$2"
  local expected_prefix="$3"
  local log_path="$4"
  local apk_path=""

  apk_path="$(installed_apk_path "${JAVA_CLIENT_PACKAGE}")"
  [ -n "${apk_path}" ] || android_die "installed APK path for ${JAVA_CLIENT_PACKAGE} not found"
  run_foreground_to_log "${log_path}" "export CLASSPATH=$(android_shell_quote "${apk_path}"); exec app_process /system/bin ${main_class} $(android_join_shell_words "${service_name}" "${expected_prefix}")"
}

start_go_server() {
  local binary_path="$1"
  local service_name="$2"
  local prefix="$3"
  local log_path="$4"
  start_background "${log_path}" "cd $(android_shell_quote "${REMOTE_DIR}") && exec ./$(basename "${binary_path}") $(android_join_shell_words "-service" "${service_name}" "-prefix" "${prefix}")"
}

start_go_baseline_server() {
  local service_name="$1"
  local log_path="$2"
  start_background "${log_path}" "cd $(android_shell_quote "${REMOTE_DIR}") && exec ./$(basename "${REMOTE_BASELINE_GO_SERVER}") $(android_join_shell_words "-service" "${service_name}")"
}

run_go_client() {
  local binary_path="$1"
  local service_name="$2"
  local expected_prefix="$3"
  local log_path="$4"
  run_foreground_to_log "${log_path}" "cd $(android_shell_quote "${REMOTE_DIR}") && exec ./$(basename "${binary_path}") $(android_join_shell_words "-service" "${service_name}" "-expect-prefix" "${expected_prefix}")"
}

run_go_matrix_client() {
  run_go_client "${REMOTE_MATRIX_GO_CLIENT}" "$1" "$2" "$3"
}

run_go_baseline_client() {
  run_go_client "${REMOTE_BASELINE_GO_CLIENT}" "$1" "$2" "$3"
}

run_case_java_server_go_client_baseline() {
  log "case: java_server_go_client baseline"
  start_java_server "${JAVA_BASELINE_SERVER_MAIN}" "${BASELINE_SERVICE_NAME}" "java" "${REMOTE_BASELINE_JAVA_SERVER_LOG}"
  run_go_baseline_client "${BASELINE_SERVICE_NAME}" "java" "${REMOTE_BASELINE_GO_CLIENT_LOG}"
  stop_last_server
}

run_case_java_server_go_client_matrix() {
  log "case: java_server_go_client matrix"
  start_java_server "${JAVA_MATRIX_SERVER_MAIN}" "${MATRIX_SERVICE_NAME}" "java" "${REMOTE_MATRIX_JAVA_SERVER_LOG}"
  run_go_matrix_client "${MATRIX_SERVICE_NAME}" "java" "${REMOTE_MATRIX_GO_CLIENT_LOG}"
  stop_last_server
}

run_case_go_server_java_client_baseline() {
  log "case: go_server_java_client baseline"
  start_go_baseline_server "${BASELINE_SERVICE_NAME}" "${REMOTE_BASELINE_GO_SERVER_LOG}"
  run_java_client "${JAVA_BASELINE_CLIENT_MAIN}" "${BASELINE_SERVICE_NAME}" "go" "${REMOTE_BASELINE_JAVA_CLIENT_LOG}"
  stop_last_server
}

run_case_go_server_java_client_matrix() {
  log "case: go_server_java_client matrix"
  start_go_server "${REMOTE_MATRIX_GO_SERVER}" "${MATRIX_SERVICE_NAME}" "go" "${REMOTE_MATRIX_GO_SERVER_LOG}"
  run_java_client "${JAVA_MATRIX_CLIENT_MAIN}" "${MATRIX_SERVICE_NAME}" "go" "${REMOTE_MATRIX_JAVA_CLIENT_LOG}"
  stop_last_server
}

main() {
  if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
  fi

  command -v "${GRADLE_BIN}" >/dev/null 2>&1 || android_die "gradle not found: ${GRADLE_BIN}"

  prepare_target
  build_java_fixtures
  install_java_fixtures
  build_go_fixtures
  push_go_fixtures

  run_case_java_server_go_client_baseline
  run_case_java_server_go_client_matrix
  run_case_go_server_java_client_baseline
  run_case_go_server_java_client_matrix

  log "all basic AIDL cases passed on ${ANDROID_SERIAL}"
}

main "$@"
