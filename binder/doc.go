// Package binder defines the draft public API for a Go-native user-space
// Binder runtime built on top of the existing Linux/Android kernel Binder
// driver.
//
// This package intentionally exposes only the API surface. Runtime internals,
// backend wiring, and driver interactions belong in internal packages.
package binder
