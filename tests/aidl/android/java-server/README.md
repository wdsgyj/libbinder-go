# Java Server Fixture Module

This module will host Java binder services used by Go fixture clients.

Planned contents:

- fixture service implementations
- callback endpoints needed by Java-originated flows
- instrumentation hooks for lifecycle control

Current baseline assets:

- `BaselineServiceImpl`
- `FixtureServiceRegistry`
- `FixtureServerMain`

The intended execution model for the first slice is:

1. build the APK
2. install it on device/emulator
3. start `FixtureServerMain` via `app_process` with the APK on `CLASSPATH`
4. let Go fixture clients resolve the service through `ServiceManager`

This keeps the first phase aligned with real binder IPC instead of falling back to a host-only mock path.
