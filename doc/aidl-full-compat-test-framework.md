# AIDL Full Compatibility Test Framework

## Goal

This framework is the acceptance boundary for the AIDL generator and binder runtime.

The target is not "Go can read what Go wrote". The target is:

- Java server -> Go client works on Android emulator and real device
- Go server -> Java client works on Android emulator and real device
- Every supported AIDL type family is covered
- Every supported invocation semantic is covered
- The result is stable under host regression, Android emulator regression, and physical-device regression

## Acceptance Rules

Coverage is considered complete only when all of the following are true:

1. Every supported AIDL type appears in both code generation and runtime interoperability tests.
2. Every supported invocation form appears in both directions when it is semantically meaningful.
3. Every case runs on Android aarch64 emulator.
4. The same matrix has a real-device execution path.
5. The host runner can build, install, execute, collect logs, and summarize results.

The matrix is intentionally not a literal Cartesian explosion. Instead, it is a coverage-complete matrix:

- every type must appear in `in`, `out`, `inout`, and return position where AIDL allows it
- every nullable type must appear in null and non-null forms
- every container type must appear with nested representative payloads
- every callback-oriented feature must appear in both registration and callback execution paths

## Matrix Dimensions

### Directions

- `java_server_go_client`
- `go_server_java_client`

### Runtime Environment

- `android_emulator_arm64`
- `android_device_arm64`

### Invocation Semantics

- synchronous call
- `oneway`
- return only
- `out`
- `inout`
- callback interface
- Android callback carrier (`ResultReceiver`, `ShellCallback`)
- binder lifecycle (`add/check/get/wait`, death recipient)
- metadata (`version/hash`, stability)

### Type Families

- primitives: `boolean`, `byte`, `char`, `int`, `long`, `float`, `double`
- `String`
- arrays
- fixed-size arrays
- `List<T>`
- `Map<K,V>`
- raw `Map`
- enum
- union
- structured parcelable
- custom / non-structured parcelable adapter
- `IBinder`
- typed interface
- `FileDescriptor`
- `ParcelFileDescriptor`
- Android framework callback carriers and parcelables

## Phase Plan

### Phase 0: Framework Foundation

- create shared AIDL fixture package
- create host runner layout
- create Android Gradle project skeleton
- create Go-side server/client fixture layout
- define machine-readable case catalog

### Phase 1: Java Server -> Go Client Primitive Baseline

- sync primitive calls
- return + `out` + `inout`
- nullable scalar/string coverage
- emulator run path

Exit criteria:

- Go client can call Java service for baseline scalar cases on emulator

### Phase 2: Go Server -> Java Client Primitive Baseline

- mirror Phase 1 in reverse direction
- service registration strategy split by environment
- real-device path for `addService`-sensitive cases

Exit criteria:

- Java client can call Go service for baseline scalar cases on emulator or documented device-only path

### Phase 3: Containers and Structured Types

- arrays
- fixed arrays
- lists
- typed maps
- raw maps
- parcelable
- enum
- union

Exit criteria:

- Java and Go agree on container wire semantics, including nested containers

### Phase 4: Binder Interfaces and Callback Flow

- `IBinder`
- typed interface argument/return
- listener registration
- callback invocation
- `oneway`

Exit criteria:

- callback round-trip works in both directions

### Phase 5: Android-Specific Callback Carriers

- `ResultReceiver`
- `ShellCallback`
- shell-command style service flows

Exit criteria:

- framework callback carriers interoperate with Java system-side expectations

### Phase 6: Lifecycle, Metadata, and Policy

- service manager `add/check/get/wait`
- death recipient
- interface version/hash
- stability label enforcement
- negative/error paths

Exit criteria:

- binder governance semantics are covered with emulator/device split where needed

### Phase 7: Transport and Scale

- RPC / unix transport on Android userspace where applicable
- large parcel payloads
- deep nesting
- repeated calls / listener churn

Exit criteria:

- no hidden runtime assumptions remain in non-trivial traffic patterns

### Phase 8: Corpus and Release Gate

- AOSP fixture corpus regression
- generated Java/Go fixture compile checks
- emulator smoke matrix
- real-device full matrix
- release gate summary

Exit criteria:

- every planned case is automated or explicitly marked blocked with reason

## Directory Layout

The framework lives under `tests/aidl`.

```text
tests/aidl/
  README.md
  cases/
    catalog.md
    catalog.json
    implementation-order.md
  host/
    README.md
  android/
    README.md
    settings.gradle.kts
    build.gradle.kts
    gradle.properties
    java-server/
    java-client/
    shared/
  go/
    README.md
    client/
    server/
    shared/
```

## Execution Model

Host runner responsibilities:

- build Go Android binaries
- build/install Java APKs or instrumentation APKs
- push fixture binaries to device
- start emulator or reuse connected device
- execute cases by ID or phase
- collect stdout, logcat, instrumentation result, and file artifacts
- emit a phase summary

Android-side responsibilities:

- Java server fixture exports binder services and callback endpoints
- Java client fixture invokes Go services and asserts results
- Go server fixture exports binder services and callback endpoints
- Go client fixture invokes Java services and asserts results

## Required Reporting

Each case must record:

- case ID
- direction
- environment
- transport
- expected result
- actual result
- artifact locations
- skip/block reason if not executed

## Relationship with Existing Scripts

Existing scripts remain useful, but they are not sufficient as the acceptance boundary:

- `scripts/android-emulator-test.sh` is still the base emulator runner
- device-specific regressions remain useful as focused probes
- the new framework becomes the place where AIDL feature-complete interoperability is tracked

## Next Milestones

1. wire the shared AIDL fixture package
2. implement Phase 1 case binaries and Java fixture service
3. wire host runner to execute a single case end-to-end on emulator
4. mirror the same case in reverse direction
5. expand by phase, not by ad hoc interface demand

## Initial Implemented Slice

The repository now contains a runnable emulator basic matrix:

- shared AIDL:
  - `BaselinePayload`
  - `IBaselineService`
  - `BasicMode`
  - `BasicUnion`
  - `BasicBundle`
  - `IBasicMatrixService`
- host runners:
  - `scripts/android-aidl-baseline-sync.sh`
  - `scripts/android-aidl-basic-cases.sh`
- emulator-passing coverage:
  - Java server -> Go client baseline
  - Go server -> Java client baseline
  - Java server -> Go client basic matrix
  - Go server -> Java client basic matrix
  - nullable string
  - `int[]`
  - fixed-size arrays
  - `List<String>`
  - `List<Parcelable>`
  - `Map<String,String>`
  - `Map<String,Parcelable>`
  - enum
  - union
  - structured parcelable
  - complex parcelable return / `out` / `inout`
- known exclusion from the Java-AIDL-driven emulator basic slice:
  - raw `Map`
    because current Android AIDL tooling rejects untyped `Map` in interface definitions
- Java server fixture:
  - `BaselineServiceImpl`
  - `FixtureServerMain`
- Java client fixture:
  - `FixtureClientMain`
- Go fixture binaries:
  - `tests/aidl/go/client/baseline`
  - `tests/aidl/go/server/baseline`
- regeneration script:
  - `scripts/update-aidl-test-generated.sh`

This slice is enough to anchor `BOOT-001` and to start implementing `SYNC-001` / `SYNC-002` on real Android targets.
