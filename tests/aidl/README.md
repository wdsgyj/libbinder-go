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

The framework now has a complete emulator matrix, a host corpus regression gate, and a wired real-device regression entry.

What exists now:

- phase plan and full case inventory
- shared fixture AIDL definitions for baseline, advanced, governance, lifecycle, scale, and extended type coverage
- Java hand-written Binder protocol shims for SDK-hidden or Java-AIDL-unsupported paths such as raw `FileDescriptor` and untyped `Map`
- Java fixture services and clients
- Go fixture services and clients
- host runner entry scripts
- emulator slice runners:
  - `basic`
  - `advanced`
  - `extended`
  - `governance`
  - `lifecycle`
  - `callbacks`
  - `scale`
  - `runtime`
- full emulator aggregation gate
- host AOSP binder corpus generate + compile gate
- real-device regression gate wiring

What is still intentionally separate:

- real-device full-matrix execution is wired, but still depends on an attached target that permits the required shell / service interactions
- some emulator fixtures continue to use hand-written Java Binder protocol shims where Android SDK AIDL tooling cannot express the required wire format

## Entry Points

- framework design: `doc/aidl-full-compat-test-framework.md`
- case catalog: `tests/aidl/cases/catalog.md`
- implementation order: `tests/aidl/cases/implementation-order.md`
- helper script: `scripts/android-aidl-matrix-test.sh`
- emulator basic matrix: `scripts/android-aidl-basic-cases.sh`
- emulator advanced matrix: `scripts/android-aidl-advanced-cases.sh`
- emulator extended matrix: `scripts/android-aidl-extended-cases.sh`
- emulator governance matrix: `scripts/android-aidl-governance-cases.sh`
- emulator lifecycle matrix: `scripts/android-aidl-lifecycle-cases.sh`
- emulator Android callback matrix: `scripts/android-aidl-android-callback-cases.sh`
- emulator scale matrix: `scripts/android-aidl-scale-cases.sh`
- emulator runtime matrix: `scripts/android-aidl-runtime-cases.sh`
- full emulator gate: `scripts/android-aidl-full-emulator.sh`
- host corpus gate: `scripts/aidl-corpus-regression.sh`
- real-device gate: `scripts/android-aidl-device-gate.sh`
