# AIDL Compatibility Test Matrix

This directory is the dedicated interoperability test framework for:

- Java server -> Go client
- Go server -> Java client
- Android emulator execution
- Android real-device execution
- AIDL feature-matrix tracking

This directory is intentionally separate from package-local unit tests.

## Layout

- `cases/`
  - case catalog and implementation order
- `host/`
  - host-side runner, build/install/orchestration conventions
- `android/`
  - Java-side fixtures, shared AIDL package, Gradle project skeleton
- `go/`
  - Go-side fixtures, shared helper conventions

## Current Status

The framework now has a runnable emulator basic matrix.

What exists now:

- phase plan and full case inventory
- shared fixture AIDL definitions for baseline + basic matrix
- Java fixture services and clients
- Go fixture services and clients
- host runner entry scripts
- emulator basic matrix runner

What still needs implementation:

- real-device execution path for custom service registration cases
- raw `Map` compatibility path outside Java AIDL's typed restrictions
- binder callback / oneway / FD / lifecycle / metadata phases
- release-gate aggregation across the full catalog

## Entry Points

- framework design: `doc/aidl-full-compat-test-framework.md`
- case catalog: `tests/aidl/cases/catalog.md`
- implementation order: `tests/aidl/cases/implementation-order.md`
- helper script: `scripts/android-aidl-matrix-test.sh`
- emulator basic matrix: `scripts/android-aidl-basic-cases.sh`
