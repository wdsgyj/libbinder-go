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
JAVA_MATRIX_SERVER_MAIN="com.wdsgyj.libbinder.aidltest.javaserver.BasicMatrixServerMain"
JAVA_MATRIX_CLIENT_MAIN="com.wdsgyj.libbinder.aidltest.javaclient.BasicMatrixClientMain"
JAVA_LISTENER_SERVER_MAIN="com.wdsgyj.libbinder.aidltest.javaserver.ListenerServerMain"
JAVA_LISTENER_CLIENT_MAIN="com.wdsgyj.libbinder.aidltest.javaclient.ListenerClientMain"

MATRIX_SERVICE_NAME="${MATRIX_SERVICE_NAME:-libbinder.go.aidltest.matrix}"
LISTENER_SERVICE_NAME="${LISTENER_SERVICE_NAME:-libbinder.go.aidltest.listener}"
CHURN_ROUNDS="${CHURN_ROUNDS:-64}"

REMOTE_MATRIX_GO_CLIENT="${REMOTE_DIR}/matrix-go-client"
REMOTE_MATRIX_GO_SERVER="${REMOTE_DIR}/matrix-go-server"
REMOTE_LISTENER_GO_CLIENT="${REMOTE_DIR}/listener-go-client"
REMOTE_LISTENER_GO_SERVER="${REMOTE_DIR}/listener-go-server"

REMOTE_MATRIX_JAVA_SERVER_LOG="${REMOTE_DIR}/scale-matrix-java-server.log"
REMOTE_MATRIX_JAVA_CLIENT_LOG="${REMOTE_DIR}/scale-matrix-java-client.log"
REMOTE_MATRIX_GO_SERVER_LOG="${REMOTE_DIR}/scale-matrix-go-server.log"
REMOTE_MATRIX_GO_CLIENT_LOG="${REMOTE_DIR}/scale-matrix-go-client.log"
REMOTE_LISTENER_JAVA_SERVER_LOG="${REMOTE_DIR}/scale-listener-java-server.log"
REMOTE_LISTENER_JAVA_CLIENT_LOG="${REMOTE_DIR}/scale-listener-java-client.log"
REMOTE_LISTENER_GO_SERVER_LOG="${REMOTE_DIR}/scale-listener-go-server.log"
REMOTE_LISTENER_GO_CLIENT_LOG="${REMOTE_DIR}/scale-listener-go-client.log"

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
  echo "[android-aidl-scale-cases] $*"
}

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-aidl-scale-cases.sh

Runs scale Android emulator AIDL cases:
  1. Java server -> Go client (large matrix payload)
  2. Go server -> Java client (large matrix payload)
  3. Java server -> Go client (listener churn)
  4. Go server -> Java client (listener churn)
EOF
}

dump_case_logs() {
  local path=""
  log "device fixture logs:"
  for path in \
    "${REMOTE_MATRIX_JAVA_SERVER_LOG}" \
    "${REMOTE_MATRIX_JAVA_CLIENT_LOG}" \
    "${REMOTE_MATRIX_GO_SERVER_LOG}" \
    "${REMOTE_MATRIX_GO_CLIENT_LOG}" \
    "${REMOTE_LISTENER_JAVA_SERVER_LOG}" \
    "${REMOTE_LISTENER_JAVA_CLIENT_LOG}" \
    "${REMOTE_LISTENER_GO_SERVER_LOG}" \
    "${REMOTE_LISTENER_GO_CLIENT_LOG}"; do
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

  emulator_log="$(mktemp "${TMPDIR:-/tmp}/libbinder-go-aidl-scale.XXXXXX")"
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
    env "${GO_ENV_ANDROID[@]}" go build -o /tmp/matrix-go-client ./tests/aidl/go/client/matrix
    env "${GO_ENV_ANDROID[@]}" go build -o /tmp/matrix-go-server ./tests/aidl/go/server/matrix
    env "${GO_ENV_ANDROID[@]}" go build -o /tmp/listener-go-client ./tests/aidl/go/client/listener
    env "${GO_ENV_ANDROID[@]}" go build -o /tmp/listener-go-server ./tests/aidl/go/server/listener
  )
}

push_go_fixtures() {
  log "pushing Go fixture binaries"
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "mkdir -p $(android_shell_quote "${REMOTE_DIR}")" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push /tmp/matrix-go-client "${REMOTE_MATRIX_GO_CLIENT}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push /tmp/matrix-go-server "${REMOTE_MATRIX_GO_SERVER}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push /tmp/listener-go-client "${REMOTE_LISTENER_GO_CLIENT}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push /tmp/listener-go-server "${REMOTE_LISTENER_GO_SERVER}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "chmod 755 $(android_shell_quote "${REMOTE_MATRIX_GO_CLIENT}") $(android_shell_quote "${REMOTE_MATRIX_GO_SERVER}") $(android_shell_quote "${REMOTE_LISTENER_GO_CLIENT}") $(android_shell_quote "${REMOTE_LISTENER_GO_SERVER}")" >/dev/null
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
  local mode="$4"
  local rounds="$5"
  local log_path="$6"
  local apk_path=""

  apk_path="$(installed_apk_path "${JAVA_CLIENT_PACKAGE}")"
  [ -n "${apk_path}" ] || android_die "installed APK path for ${JAVA_CLIENT_PACKAGE} not found"
  run_foreground_to_log "${log_path}" "export CLASSPATH=$(android_shell_quote "${apk_path}"); exec app_process /system/bin ${main_class} $(android_join_shell_words "${service_name}" "${expected_prefix}" "${mode}" "${rounds}")"
}

run_go_client() {
  local binary_path="$1"
  local service_name="$2"
  local expected_prefix="$3"
  local mode="$4"
  local rounds="$5"
  local log_path="$6"
  local args=("-service" "${service_name}" "-expect-prefix" "${expected_prefix}" "-mode" "${mode}")
  if [ "${rounds}" -gt 0 ] 2>/dev/null; then
    args+=("-rounds" "${rounds}")
  fi
  run_foreground_to_log "${log_path}" "cd $(android_shell_quote "${REMOTE_DIR}") && exec ./$(basename "${binary_path}") $(android_join_shell_words "${args[@]}")"
}

start_go_server() {
  local binary_path="$1"
  local service_name="$2"
  local prefix="$3"
  local log_path="$4"
  start_background "${log_path}" "cd $(android_shell_quote "${REMOTE_DIR}") && exec ./$(basename "${binary_path}") $(android_join_shell_words "-service" "${service_name}" "-prefix" "${prefix}")"
}

run_case_java_server_go_client_matrix_perf() {
  log "case: java_server_go_client matrix perf"
  start_java_server "${JAVA_MATRIX_SERVER_MAIN}" "${MATRIX_SERVICE_NAME}" "java" "${REMOTE_MATRIX_JAVA_SERVER_LOG}"
  run_go_client "${REMOTE_MATRIX_GO_CLIENT}" "${MATRIX_SERVICE_NAME}" "java" "perf" "0" "${REMOTE_MATRIX_GO_CLIENT_LOG}"
  stop_last_server
}

run_case_go_server_java_client_matrix_perf() {
  log "case: go_server_java_client matrix perf"
  start_go_server "${REMOTE_MATRIX_GO_SERVER}" "${MATRIX_SERVICE_NAME}" "go" "${REMOTE_MATRIX_GO_SERVER_LOG}"
  run_java_client "${JAVA_MATRIX_CLIENT_MAIN}" "${MATRIX_SERVICE_NAME}" "go" "perf" "0" "${REMOTE_MATRIX_JAVA_CLIENT_LOG}"
  stop_last_server
}

run_case_java_server_go_client_listener_churn() {
  log "case: java_server_go_client listener churn"
  start_java_server "${JAVA_LISTENER_SERVER_MAIN}" "${LISTENER_SERVICE_NAME}" "_" "${REMOTE_LISTENER_JAVA_SERVER_LOG}"
  run_go_client "${REMOTE_LISTENER_GO_CLIENT}" "${LISTENER_SERVICE_NAME}" "_" "churn" "${CHURN_ROUNDS}" "${REMOTE_LISTENER_GO_CLIENT_LOG}"
  stop_last_server
}

run_case_go_server_java_client_listener_churn() {
  log "case: go_server_java_client listener churn"
  start_go_server "${REMOTE_LISTENER_GO_SERVER}" "${LISTENER_SERVICE_NAME}" "_" "${REMOTE_LISTENER_GO_SERVER_LOG}"
  run_java_client "${JAVA_LISTENER_CLIENT_MAIN}" "${LISTENER_SERVICE_NAME}" "_" "churn" "${CHURN_ROUNDS}" "${REMOTE_LISTENER_JAVA_CLIENT_LOG}"
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

  run_case_java_server_go_client_matrix_perf
  run_case_go_server_java_client_matrix_perf
  run_case_java_server_go_client_listener_churn
  run_case_go_server_java_client_listener_churn

  log "all scale AIDL cases passed on ${ANDROID_SERIAL}"
}

main "$@"
