# Go Fixtures

This subtree will host Go-side interoperability fixtures.

Subdirectories:

- `client/`
  - Go binaries that call Java fixture services
- `server/`
  - Go binaries that export binder services for Java fixture clients
- `shared/`
  - Go helper code shared by both directions

The fixtures here are intentionally separate from package-local unit tests.
