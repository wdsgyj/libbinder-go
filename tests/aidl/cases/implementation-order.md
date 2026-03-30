# AIDL Compatibility Implementation Order

## Step 1: Framework Foundation

- BOOT-001
- BOOT-002
- BOOT-003
- BOOT-004

Deliverables:

- shared fixture package layout
- host runner skeleton
- Android project skeleton
- Go fixture skeleton
- artifact directory convention

## Step 2: Baseline Java Server -> Go Client

- SYNC-001
- DIR-001
- NULL-001

Deliverables:

- Java fixture service APK
- Go client fixture binary
- one emulator-run happy path

## Step 3: Baseline Go Server -> Java Client

- SYNC-002
- DIR-002
- NULL-002

Deliverables:

- Go fixture service binary
- Java client instrumentation
- environment split for service-manager-sensitive cases

## Step 4: Structured Types and Containers

- PARC-001
- MAP-001
- MAP-002
- MAP-003
- ARR-001
- LIST-001
- ENUM-001
- UNION-001

Deliverables:

- typed fixture AIDLs for representative containers
- nested payload assertions in both directions

## Step 5: Extended Container and Parcelable Semantics

- ARR-002
- LIST-002
- LIST-003
- MAP-004
- PARC-002
- PARC-003

Deliverables:

- nested list/map coverage
- custom parcelable adapter coverage
- default value / nullable field coverage

## Step 6: Binder Interface and Oneway Flow

- BIND-001
- BIND-002
- BIND-003
- ONEW-001

Deliverables:

- callback registration
- callback delivery
- oneway ordering assertions

## Step 7: Android-Specific Callback Carriers

- ANDR-001
- ANDR-002

Deliverables:

- `ResultReceiver` fixture path
- `ShellCallback` fixture path

## Step 8: Lifecycle, Metadata, and Negative Paths

- META-001
- META-002
- LIFE-001
- LIFE-002
- EXC-001
- EXC-002
- EXC-003
- FD-001
- FD-002

Deliverables:

- service manager and death tests
- stability / version / hash assertions
- binder error-path assertions
- FD ownership assertions

## Step 9: Scale and Alternate Transport

- RPC-001
- PERF-001
- PERF-002

Deliverables:

- Android userspace RPC transport regression
- large-payload and churn scenarios

## Step 10: Release Gate

- REAL-001
- REAL-002
- CORP-001

Deliverables:

- emulator release command
- real-device release command
- AOSP corpus regression command

## Recommended First Executable Slice

The smallest meaningful end-to-end slice is:

1. BOOT-001
2. BOOT-002
3. BOOT-003
4. SYNC-001
5. DIR-001
6. SYNC-002
7. DIR-002
8. PARC-001
9. MAP-001
10. BIND-002

That slice proves:

- both directions exist
- basic scalar and parcelable codecs work
- maps are interoperable
- callback interface round-trip exists
