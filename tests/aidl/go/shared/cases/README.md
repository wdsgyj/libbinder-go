# Shared Go Case Helpers

This package area is reserved for helper code reused by multiple Go-side fixture binaries.

It also contains host-side regression tests for Binder object transport that does not depend on
`ServiceManager` registration, including:

- raw `IBinder` argument/return round-trip
- typed callback interface argument round-trip
- passing back an unregistered AIDL server binder and invoking its typed methods from Go
