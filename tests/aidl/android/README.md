# Android Java Fixtures

This subtree is the Java-side interoperability harness.

Modules:

- `shared/`
  - shared AIDL fixtures and common Java helpers
- `java-server/`
  - Java fixture services to be called by Go clients
- `java-client/`
  - Java fixture clients / instrumentation to call Go services

The project is intentionally scoped to Android only.
