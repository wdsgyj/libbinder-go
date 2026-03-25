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
ANDROID_EMULATOR_PORT="${ANDROID_EMULATOR_PORT:-5560}"
REMOTE_TEST_DIR="${REMOTE_TEST_DIR:-/data/local/tmp/libbinder-go-tests}"
ANDROID_SERIAL="${ANDROID_SERIAL:-emulator-${ANDROID_EMULATOR_PORT}}"

GO_ENV_ANDROID=(GOOS=android GOARCH=arm64 CGO_ENABLED=0)
TEST_PACKAGES=()
TEST_BINARY_ARGS=("-test.v")
EMULATOR_LOG=""
TMP_DIR=""

cleanup() {
  local exit_code=$?

  if [ "${ANDROID_EMULATOR_STARTED:-0}" = "1" ]; then
    android_stop_emulator "${ANDROID_SERIAL}"
  fi

  if [ "${exit_code}" -ne 0 ] && [ -n "${EMULATOR_LOG}" ] && [ -f "${EMULATOR_LOG}" ]; then
    echo "emulator log: ${EMULATOR_LOG}" >&2
  elif [ -n "${TMP_DIR}" ] && [ -d "${TMP_DIR}" ]; then
    rm -rf "${TMP_DIR}"
  fi

  exit "${exit_code}"
}
trap cleanup EXIT

usage() {
  cat <<'EOF'
Usage:
  ./scripts/android-emulator-test.sh [go package patterns...] [-- test-binary-args...]

Examples:
  ./scripts/android-emulator-test.sh
  ./scripts/android-emulator-test.sh ./binder ./internal/protocol
  ./scripts/android-emulator-test.sh ./internal/kernel -- -test.v -test.run TestDriverManagerOpenCloseOnAndroid

Environment:
  ANDROID_SDK_ROOT              Android SDK root. Auto-detected if unset.
  ANDROID_API_LEVEL             Android API level. Default: 35.
  ANDROID_IMAGE_TAG             System image tag. Default: google_apis.
  ANDROID_ABI                   Emulator ABI. Default: arm64-v8a.
  ANDROID_AVD_NAME              AVD name. Auto-created if missing.
  ANDROID_DEVICE_PROFILE        avdmanager device profile. Default: medium_phone.
  ANDROID_EMULATOR_PORT         Emulator console/adb port. Default: 5560.
  ANDROID_SERIAL                Existing device/emulator serial to reuse.
  ANDROID_SKIP_SDK_INSTALL      Set to 1 to forbid automatic sdkmanager installs.
  ANDROID_ADB_ROOT              Set to 1 to run tests after `adb root`. Default: 1.
  ANDROID_TEST_AS_ROOT          Set to 1 to execute test binaries via `su 0` when available. Default: 1.
  ANDROID_HEADLESS              Set to 0 to show the emulator window.
  ANDROID_WIPE_DATA             Set to 1 to boot with -wipe-data.
  ANDROID_KEEP_EMULATOR         Set to 1 to leave the emulator running.
  ANDROID_EMULATOR_EXTRA_ARGS   Extra raw emulator args, appended as-is.
  REMOTE_TEST_DIR               Device-side temp dir. Default: /data/local/tmp/libbinder-go-tests.
EOF
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      TEST_BINARY_ARGS=("$@")
      break
      ;;
    *)
      TEST_PACKAGES+=("$1")
      ;;
  esac
  shift
done

if [ "${#TEST_PACKAGES[@]}" -eq 0 ]; then
  TEST_PACKAGES=("./...")
fi

android_setup_paths

if android_device_online "${ANDROID_SERIAL}"; then
  android_log "using already connected device ${ANDROID_SERIAL}"
elif [ -d "$(android_avd_dir)" ]; then
  android_log "using existing AVD ${ANDROID_AVD_NAME}"
else
  android_ensure_sdk_components
  android_ensure_avd
fi

TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/libbinder-go-android-tests.XXXXXX")"
EMULATOR_LOG="${TMP_DIR}/emulator.log"

android_start_emulator "${ANDROID_SERIAL}" "${ANDROID_EMULATOR_PORT}" "${EMULATOR_LOG}"

if [ "${ANDROID_ADB_ROOT:-1}" = "1" ]; then
  android_root_device "${ANDROID_SERIAL}"
fi

android_log "cross-compiling module for android/arm64"
(
  cd "${ROOT_DIR}"
  env "${GO_ENV_ANDROID[@]}" go build "${TEST_PACKAGES[@]}"
)

android_log "preparing remote test directory ${REMOTE_TEST_DIR}"
"${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "rm -rf $(android_shell_quote "${REMOTE_TEST_DIR}") && mkdir -p $(android_shell_quote "${REMOTE_TEST_DIR}")" >/dev/null

TEST_IMPORT_PATHS=()
while IFS= read -r pkg; do
  if [ -n "${pkg}" ]; then
    TEST_IMPORT_PATHS+=("${pkg}")
  fi
done < <(
  cd "${ROOT_DIR}" && \
  env "${GO_ENV_ANDROID[@]}" go list -f '{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}{{end}}' "${TEST_PACKAGES[@]}"
)

if [ "${#TEST_IMPORT_PATHS[@]}" -eq 0 ]; then
  android_log "no Go test packages matched; android build smoke passed"
  exit 0
fi

for pkg in "${TEST_IMPORT_PATHS[@]}"; do
  test_name="$(printf '%s' "${pkg}" | sed 's#[^A-Za-z0-9_.-]#_#g').test"
  host_test_bin="${TMP_DIR}/${test_name}"
  remote_test_bin="${REMOTE_TEST_DIR}/${test_name}"

  android_log "building test binary for ${pkg}"
  (
    cd "${ROOT_DIR}"
    env "${GO_ENV_ANDROID[@]}" go test -c -o "${host_test_bin}" "${pkg}"
  )

  android_log "pushing ${test_name} to emulator"
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" push "${host_test_bin}" "${remote_test_bin}" >/dev/null
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "chmod 755 $(android_shell_quote "${remote_test_bin}")" >/dev/null

  remote_cmd="cd $(android_shell_quote "${REMOTE_TEST_DIR}") && exec $(android_join_shell_words "./${test_name}" "${TEST_BINARY_ARGS[@]}")"
  if [ "${ANDROID_TEST_AS_ROOT:-1}" = "1" ] && "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "command -v su >/dev/null 2>&1"; then
    remote_cmd="exec su 0 sh -c $(android_shell_quote "${remote_cmd}")"
    android_log "running ${pkg} on ${ANDROID_SERIAL} via su 0"
  else
    android_log "running ${pkg} on ${ANDROID_SERIAL}"
  fi
  "${ADB_BIN}" -s "${ANDROID_SERIAL}" shell "${remote_cmd}"
done

android_log "all android emulator tests passed"
