# AIDL Compatibility Case Catalog

## Coverage Policy

The goal is feature-complete coverage, not an unbounded Cartesian product.

Each type family must appear in:

- Java server -> Go client
- Go server -> Java client
- success path
- null / empty / boundary path where allowed

Each invocation semantic must appear in:

- normal call
- representative negative path

## Case List

| ID | Phase | Priority | Directions | Area | Summary |
| --- | --- | --- | --- | --- | --- |
| BOOT-001 | 0 | P0 | both | framework | Shared fixture AIDL package can be consumed by Java and Go |
| BOOT-002 | 0 | P0 | both | framework | Host runner builds APKs and Go binaries and installs them on target |
| BOOT-003 | 0 | P0 | both | framework | Emulator execution path is wired and artifact collection works |
| BOOT-004 | 0 | P0 | both | framework | Real-device execution path is wired and artifact collection works |
| SYNC-001 | 1 | P0 | java_server_go_client | call | Primitive synchronous call baseline |
| SYNC-002 | 2 | P0 | go_server_java_client | call | Primitive synchronous call baseline |
| DIR-001 | 1 | P0 | java_server_go_client | call | Return + `out` + `inout` scalar semantics |
| DIR-002 | 2 | P0 | go_server_java_client | call | Return + `out` + `inout` scalar semantics |
| NULL-001 | 1 | P1 | java_server_go_client | type | Nullable `String` and nullable parcelable |
| NULL-002 | 2 | P1 | go_server_java_client | type | Nullable `String` and nullable parcelable |
| ARR-001 | 3 | P1 | both | type | Primitive arrays |
| ARR-002 | 3 | P1 | both | type | Fixed-size arrays |
| LIST-001 | 3 | P1 | both | type | `List<int>` / `List<String>` |
| LIST-002 | 3 | P1 | both | type | `List<Parcelable>` |
| LIST-003 | 3 | P2 | both | type | Nested lists |
| MAP-001 | 3 | P0 | both | type | `Map<String,String>` |
| MAP-002 | 3 | P0 | both | type | `Map<String,Parcelable>` |
| MAP-003 | 3 | P0 | both | type | raw `Map` dynamic values |
| MAP-004 | 3 | P1 | both | type | nested `Map` + `List` combinations |
| ENUM-001 | 3 | P1 | both | type | enum argument / return / nested enum |
| UNION-001 | 3 | P1 | both | type | union argument / return / nested union |
| PARC-001 | 3 | P0 | both | type | structured parcelable baseline |
| PARC-002 | 3 | P1 | both | type | nested parcelable with defaults and nullable fields |
| PARC-003 | 3 | P2 | both | type | custom / non-structured parcelable adapter |
| BIND-001 | 4 | P0 | both | binder | raw `IBinder` argument and return |
| BIND-002 | 4 | P0 | both | binder | typed callback interface round-trip |
| BIND-003 | 4 | P1 | both | binder | listener registration and repeated callbacks |
| ONEW-001 | 4 | P0 | both | call | `oneway` transaction semantics |
| EXC-001 | 6 | P1 | both | error | Remote exception propagation |
| EXC-002 | 6 | P1 | both | error | Unknown transaction and unsupported path |
| EXC-003 | 6 | P1 | both | error | Null binder handling and null callback removal |
| FD-001 | 6 | P1 | both | type | `FileDescriptor` transport |
| FD-002 | 6 | P1 | both | type | `ParcelFileDescriptor` transport |
| ANDR-001 | 5 | P0 | both | android | `ResultReceiver` callback flow |
| ANDR-002 | 5 | P0 | both | android | `ShellCallback` callback flow |
| META-001 | 6 | P0 | both | metadata | interface version/hash |
| META-002 | 6 | P0 | both | metadata | stability label enforcement and partition semantics |
| LIFE-001 | 6 | P0 | both | lifecycle | service manager `add/check/get/wait` |
| LIFE-002 | 6 | P1 | both | lifecycle | death recipient |
| RPC-001 | 7 | P2 | both | transport | RPC / unix transport on Android userspace |
| PERF-001 | 7 | P2 | both | scale | large parcel payload |
| PERF-002 | 7 | P2 | both | scale | deep nesting and repeated callback churn |
| REAL-001 | 8 | P0 | both | release | Emulator full matrix release gate |
| REAL-002 | 8 | P0 | both | release | Real-device full matrix release gate |
| CORP-001 | 8 | P0 | both | release | AOSP fixture corpus parse/generate/compile regression |

## Notes

- Cases marked `P0` are release-blocking.
- Cases marked `P1` should be completed before claiming "feature-complete interoperability".
- Cases marked `P2` are still required for the full target, but can follow after the blocking path is stable.
- Current emulator-complete basic slice:
  - `BOOT-001`
  - `BOOT-002`
  - `BOOT-003`
  - `SYNC-001`
  - `SYNC-002`
  - `DIR-001`
  - `DIR-002`
  - `NULL-001`
  - `NULL-002`
  - `ARR-001`
  - `ARR-002`
  - `LIST-001`
  - `LIST-002`
  - `MAP-001`
  - `MAP-002`
  - `ENUM-001`
  - `UNION-001`
  - `PARC-001`
- Current emulator-complete advanced slice:
  - `BIND-001`
  - `BIND-002`
  - `ONEW-001`
  - `EXC-001`
  - `FD-001`
  - `FD-002`
- `MAP-003` is intentionally not part of the Java-AIDL-driven basic emulator matrix because current Android AIDL tooling rejects untyped `Map` in interface definitions.
- `FD-001` cannot be exercised through a normal Java AIDL SDK stub because `FileDescriptor` marshaling in generated Java uses hidden `Parcel.readRawFileDescriptor()` / `writeRawFileDescriptor()`. The emulator fixture therefore keeps the Go side on generated AIDL and uses a hand-written Java Binder protocol shim with the same descriptor and transaction codes.
