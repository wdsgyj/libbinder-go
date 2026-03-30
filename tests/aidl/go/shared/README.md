# Go Shared Fixtures

This subtree contains Go-side shared protocol assets for the AIDL compatibility matrix.

Subdirectories:

- `generated/`
  - checked-in Go bindings generated from `tests/aidl/android/shared/src/main/aidl`
- `cases/`
  - shared structs and helper code for concrete fixture cases

Regeneration entry point:

- `scripts/update-aidl-test-generated.sh`
