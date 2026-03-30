# Java Client Fixture Module

This module will host Java-side clients and instrumentation tests that call Go fixture services.

Planned contents:

- Java client wrappers
- instrumentation assertions
- result reporting hooks for host orchestration

Current baseline assets:

- `FixtureServiceLookup`
- `FixtureClientMain`

The first executable reverse-direction slice will use the same `ServiceManager` lookup pattern as the Java server module.
