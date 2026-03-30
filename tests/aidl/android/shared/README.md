# Shared Android Fixture Package

This module will host:

- shared fixture AIDL files
- shared Java helpers
- shared parcelable and callback fixtures

Its contract is to stay small and stable so both `java-server` and `java-client` consume the same protocol definitions.
