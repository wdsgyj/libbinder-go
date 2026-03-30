# Host Runner

This directory is reserved for host-side orchestration assets.

Planned responsibilities:

- build Go Android binaries
- build Java APKs / instrumentation APKs
- install artifacts on emulator or physical device
- select cases by ID, phase, direction, or environment
- collect logs and summarize results

The initial entry script is:

- `scripts/android-aidl-matrix-test.sh`

Current executable slice script:

- `scripts/android-aidl-baseline-sync.sh`

That script is the first end-to-end runner for:

- Java server -> Go client
- `IBaselineService`
- emulator or connected device
