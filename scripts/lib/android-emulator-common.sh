#!/usr/bin/env bash

set -euo pipefail

android_die() {
  echo "error: $*" >&2
  exit 1
}

android_log() {
  echo "[android-emulator] $*"
}

android_default_sdk_root() {
  if [ -n "${ANDROID_SDK_ROOT:-}" ] && [ -d "${ANDROID_SDK_ROOT}" ]; then
    printf '%s\n' "${ANDROID_SDK_ROOT}"
    return 0
  fi
  if [ -n "${ANDROID_HOME:-}" ] && [ -d "${ANDROID_HOME}" ]; then
    printf '%s\n' "${ANDROID_HOME}"
    return 0
  fi
  if [ -d "${HOME}/Library/Android/sdk" ]; then
    printf '%s\n' "${HOME}/Library/Android/sdk"
    return 0
  fi
  if [ -d "${HOME}/Android/Sdk" ]; then
    printf '%s\n' "${HOME}/Android/Sdk"
    return 0
  fi
  return 1
}

android_shell_quote() {
  printf "'%s'" "$(printf "%s" "$1" | sed "s/'/'\\\\''/g")"
}

android_join_shell_words() {
  local out=""
  local word=""

  for word in "$@"; do
    if [ -n "${out}" ]; then
      out="${out} "
    fi
    out="${out}$(android_shell_quote "${word}")"
  done

  printf '%s' "${out}"
}

android_setup_paths() {
  ANDROID_SDK_ROOT="${ANDROID_SDK_ROOT:-$(android_default_sdk_root || true)}"
  [ -n "${ANDROID_SDK_ROOT}" ] || android_die "ANDROID_SDK_ROOT is not set and no Android SDK was found"
  [ -d "${ANDROID_SDK_ROOT}" ] || android_die "Android SDK root does not exist: ${ANDROID_SDK_ROOT}"
  export ANDROID_SDK_ROOT

  ADB_BIN="${ADB_BIN:-${ANDROID_SDK_ROOT}/platform-tools/adb}"
  EMULATOR_BIN="${EMULATOR_BIN:-${ANDROID_SDK_ROOT}/emulator/emulator}"
  SDKMANAGER_BIN="${SDKMANAGER_BIN:-${ANDROID_SDK_ROOT}/cmdline-tools/latest/bin/sdkmanager}"
  AVDMANAGER_BIN="${AVDMANAGER_BIN:-${ANDROID_SDK_ROOT}/cmdline-tools/latest/bin/avdmanager}"

  [ -x "${ADB_BIN}" ] || android_die "adb not found: ${ADB_BIN}"
  [ -x "${EMULATOR_BIN}" ] || android_die "emulator not found: ${EMULATOR_BIN}"
  [ -x "${SDKMANAGER_BIN}" ] || android_die "sdkmanager not found: ${SDKMANAGER_BIN}"
  [ -x "${AVDMANAGER_BIN}" ] || android_die "avdmanager not found: ${AVDMANAGER_BIN}"
}

android_system_image_package() {
  printf 'system-images;android-%s;%s;%s\n' \
    "${ANDROID_API_LEVEL}" \
    "${ANDROID_IMAGE_TAG}" \
    "${ANDROID_ABI}"
}

android_platform_package() {
  printf 'platforms;android-%s\n' "${ANDROID_API_LEVEL}"
}

android_system_image_dir() {
  printf '%s/system-images/android-%s/%s/%s\n' \
    "${ANDROID_SDK_ROOT}" \
    "${ANDROID_API_LEVEL}" \
    "${ANDROID_IMAGE_TAG}" \
    "${ANDROID_ABI}"
}

android_avd_dir() {
  local avd_home="${ANDROID_AVD_HOME:-${HOME}/.android/avd}"
  printf '%s/%s.avd\n' "${avd_home}" "${ANDROID_AVD_NAME}"
}

android_device_online() {
  local serial="$1"
  local state=""

  if ! state="$("${ADB_BIN}" -s "${serial}" get-state 2>/dev/null)"; then
    return 1
  fi

  [ "${state}" = "device" ]
}

android_ensure_sdk_components() {
  local system_image_pkg=""
  local platform_pkg=""
  local need_install=0
  local component_args=()

  system_image_pkg="$(android_system_image_package)"
  platform_pkg="$(android_platform_package)"

  if [ ! -x "${ANDROID_SDK_ROOT}/platform-tools/adb" ]; then
    need_install=1
    component_args+=("platform-tools")
  fi
  if [ ! -x "${ANDROID_SDK_ROOT}/emulator/emulator" ]; then
    need_install=1
    component_args+=("emulator")
  fi
  if [ ! -d "${ANDROID_SDK_ROOT}/platforms/android-${ANDROID_API_LEVEL}" ]; then
    need_install=1
    component_args+=("${platform_pkg}")
  fi
  if [ ! -d "$(android_system_image_dir)" ]; then
    need_install=1
    component_args+=("${system_image_pkg}")
  fi

  if [ "${need_install}" -eq 0 ]; then
    return 0
  fi

  if [ "${ANDROID_SKIP_SDK_INSTALL:-0}" = "1" ]; then
    android_die "required SDK components are missing and ANDROID_SKIP_SDK_INSTALL=1"
  fi

  android_log "installing SDK components: ${component_args[*]}"
  yes | "${SDKMANAGER_BIN}" --install "${component_args[@]}" >/dev/null
}

android_ensure_avd() {
  local avd_path=""
  local system_image_pkg=""

  avd_path="$(android_avd_dir)"
  if [ -d "${avd_path}" ]; then
    return 0
  fi

  system_image_pkg="$(android_system_image_package)"
  android_log "creating AVD ${ANDROID_AVD_NAME} from ${system_image_pkg}"
  mkdir -p "$(dirname "${avd_path}")"
  printf 'no\n' | "${AVDMANAGER_BIN}" create avd \
    --force \
    --name "${ANDROID_AVD_NAME}" \
    --abi "${ANDROID_ABI}" \
    --device "${ANDROID_DEVICE_PROFILE}" \
    --package "${system_image_pkg}" >/dev/null
}

android_wait_for_boot() {
  local serial="$1"
  local timeout_secs="${ANDROID_BOOT_TIMEOUT_SECS:-420}"
  local start_ts=""
  local boot_completed=""
  local dev_bootcomplete=""

  start_ts="$(date +%s)"
  "${ADB_BIN}" -s "${serial}" wait-for-device >/dev/null

  while true; do
    boot_completed="$("${ADB_BIN}" -s "${serial}" shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')"
    dev_bootcomplete="$("${ADB_BIN}" -s "${serial}" shell getprop dev.bootcomplete 2>/dev/null | tr -d '\r')"
    if [ "${boot_completed}" = "1" ] || [ "${dev_bootcomplete}" = "1" ]; then
      break
    fi

    if [ $(( $(date +%s) - start_ts )) -ge "${timeout_secs}" ]; then
      android_die "emulator ${serial} did not finish booting within ${timeout_secs}s"
    fi
    sleep 2
  done

  "${ADB_BIN}" -s "${serial}" shell input keyevent 82 >/dev/null 2>&1 || true
}

android_start_emulator() {
  local serial="$1"
  local port="$2"
  local log_file="$3"
  local emulator_args=()

  if android_device_online "${serial}"; then
    android_log "reusing already running emulator ${serial}"
    ANDROID_EMULATOR_STARTED=0
    return 0
  fi

  emulator_args=(
    "@${ANDROID_AVD_NAME}"
    "-port" "${port}"
    "-no-snapshot"
    "-no-snapshot-save"
    "-netdelay" "none"
    "-netspeed" "full"
    "-no-boot-anim"
    "-no-audio"
  )

  if [ "${ANDROID_HEADLESS:-1}" = "1" ]; then
    emulator_args+=("-no-window")
  fi
  if [ "${ANDROID_WIPE_DATA:-0}" = "1" ]; then
    emulator_args+=("-wipe-data")
  fi
  if [ -n "${ANDROID_EMULATOR_EXTRA_ARGS:-}" ]; then
    # shellcheck disable=SC2206
    emulator_args+=(${ANDROID_EMULATOR_EXTRA_ARGS})
  fi

  android_log "starting emulator ${serial}; log: ${log_file}"
  "${EMULATOR_BIN}" "${emulator_args[@]}" >"${log_file}" 2>&1 &
  ANDROID_EMULATOR_PID=$!
  ANDROID_EMULATOR_STARTED=1

  android_wait_for_boot "${serial}"
}

android_stop_emulator() {
  local serial="$1"

  if [ "${ANDROID_KEEP_EMULATOR:-0}" = "1" ]; then
    android_log "keeping emulator ${serial} running"
    return 0
  fi

  if android_device_online "${serial}"; then
    android_log "stopping emulator ${serial}"
    "${ADB_BIN}" -s "${serial}" emu kill >/dev/null 2>&1 || true
  fi

  if [ -n "${ANDROID_EMULATOR_PID:-}" ]; then
    wait "${ANDROID_EMULATOR_PID}" 2>/dev/null || true
  fi
}
