# Go Server Fixtures

Planned contents:

- per-case Go binder services
- callback endpoints
- service-manager registration helpers
- environment-aware startup strategy for emulator vs device

Current baseline asset:

- `baseline/main.go`

This binary exports `IBaselineService` through the Go runtime and is intended to be consumed by the Java fixture client path.
