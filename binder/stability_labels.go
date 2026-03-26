package binder

import (
	"context"
	"fmt"
)

// StabilityLevel identifies the Binder stability contract attached to a Binder object.
type StabilityLevel int32

const (
	StabilityUndeclared StabilityLevel = 0
	StabilityVendor     StabilityLevel = 0b000011
	StabilitySystem     StabilityLevel = 0b001100
	StabilityVINTF      StabilityLevel = 0b111111
)

// StabilityProvider exposes a Binder or Handler stability label.
type StabilityProvider interface {
	StabilityLevel() StabilityLevel
}

// ParcelBinderStabilityMarshaler writes a Binder into a Parcel using the supplied stability.
type ParcelBinderStabilityMarshaler interface {
	WriteBinderToParcelWithStability(p *Parcel, level StabilityLevel) error
}

type requiredStabilityContextKey struct{}

func DefaultLocalStability() StabilityLevel {
	return StabilitySystem
}

func (l StabilityLevel) String() string {
	switch l {
	case StabilityUndeclared:
		return "undeclared"
	case StabilityVendor:
		return "vendor"
	case StabilitySystem:
		return "system"
	case StabilityVINTF:
		return "vintf"
	default:
		return "unknown"
	}
}

func (l StabilityLevel) IsDeclared() bool {
	switch l {
	case StabilityVendor, StabilitySystem, StabilityVINTF:
		return true
	default:
		return false
	}
}

// CheckStability reports whether a provided Binder stability satisfies the required stability.
func CheckStability(provided, required StabilityLevel) bool {
	if required == StabilityUndeclared {
		return true
	}
	if !required.IsDeclared() || !provided.IsDeclared() {
		return false
	}
	return int32(provided)&int32(required) == int32(required)
}

// WithRequiredStability overrides the required stability context for outgoing Binder calls.
func WithRequiredStability(ctx context.Context, level StabilityLevel) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, requiredStabilityContextKey{}, level)
}

// RequiredStabilityFromContext reports a per-call stability override from ctx.
func RequiredStabilityFromContext(ctx context.Context) (StabilityLevel, bool) {
	if ctx == nil {
		return StabilityUndeclared, false
	}
	level, ok := ctx.Value(requiredStabilityContextKey{}).(StabilityLevel)
	return level, ok
}

// RequiredStabilityForTransact resolves the required stability for a transaction.
func RequiredStabilityForTransact(ctx context.Context, flags Flags, defaultLevel StabilityLevel) StabilityLevel {
	if flags&FlagPrivateVendor != 0 {
		return StabilityVendor
	}
	if level, ok := RequiredStabilityFromContext(ctx); ok && level != StabilityUndeclared {
		return level
	}
	if defaultLevel == StabilityUndeclared {
		return DefaultLocalStability()
	}
	return defaultLevel
}

// PrepareTransactFlags strips userspace-only flags before hitting the transport.
func PrepareTransactFlags(flags Flags) Flags {
	return flags &^ FlagPrivateVendor
}

// EnforceTransactStability validates that a user transaction is legal for the current context.
func EnforceTransactStability(ctx context.Context, target Binder, code uint32, flags Flags, defaultLevel StabilityLevel) error {
	if code < FirstCallTransaction || code > LastCallTransaction {
		return nil
	}
	required := RequiredStabilityForTransact(ctx, flags, defaultLevel)
	if required == StabilityUndeclared {
		return nil
	}
	provided := BinderStability(target)
	if CheckStability(provided, required) {
		return nil
	}
	return fmt.Errorf("%w: binder stability %s does not satisfy required %s", &StatusCodeError{Code: StatusBadType}, provided, required)
}

// BinderStability returns the declared stability label for a Binder.
func BinderStability(b Binder) StabilityLevel {
	if provider, ok := b.(StabilityProvider); ok {
		level := provider.StabilityLevel()
		if level != StabilityUndeclared {
			return level
		}
	}
	return DefaultLocalStability()
}

// HandlerStability returns the declared stability label for a local Handler.
func HandlerStability(h Handler) StabilityLevel {
	if provider, ok := h.(StabilityProvider); ok {
		level := provider.StabilityLevel()
		if level != StabilityUndeclared {
			return level
		}
	}
	return DefaultLocalStability()
}

// WithStability annotates a Handler with a Binder stability label while preserving
// stable-AIDL version/hash providers when present.
func WithStability(handler Handler, level StabilityLevel) Handler {
	switch h := handler.(type) {
	case nil:
		return nil
	case interface {
		Handler
		InterfaceVersionProvider
		InterfaceHashProvider
	}:
		return stableVersionHashHandler{handler: h, level: level}
	case interface {
		Handler
		InterfaceVersionProvider
	}:
		return stableVersionHandler{handler: h, level: level}
	case interface {
		Handler
		InterfaceHashProvider
	}:
		return stableHashHandler{handler: h, level: level}
	default:
		return stableHandler{handler: handler, level: level}
	}
}

func ForceDowngradeToLocalStability(handler Handler) Handler {
	return WithStability(handler, DefaultLocalStability())
}

func ForceDowngradeToSystemStability(handler Handler) Handler {
	return WithStability(handler, StabilitySystem)
}

func ForceDowngradeToVendorStability(handler Handler) Handler {
	return WithStability(handler, StabilityVendor)
}

func RequiresVINTFDeclaration(level StabilityLevel) bool {
	return CheckStability(level, StabilityVINTF)
}

type stableHandler struct {
	handler Handler
	level   StabilityLevel
}

func (h stableHandler) Descriptor() string {
	return h.handler.Descriptor()
}

func (h stableHandler) HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
	return h.handler.HandleTransact(ctx, code, data)
}

func (h stableHandler) StabilityLevel() StabilityLevel {
	return h.level
}

type stableVersionHandler struct {
	handler interface {
		Handler
		InterfaceVersionProvider
	}
	level StabilityLevel
}

func (h stableVersionHandler) Descriptor() string {
	return h.handler.Descriptor()
}

func (h stableVersionHandler) HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
	return h.handler.HandleTransact(ctx, code, data)
}

func (h stableVersionHandler) InterfaceVersion() int32 {
	return h.handler.InterfaceVersion()
}

func (h stableVersionHandler) StabilityLevel() StabilityLevel {
	return h.level
}

type stableHashHandler struct {
	handler interface {
		Handler
		InterfaceHashProvider
	}
	level StabilityLevel
}

func (h stableHashHandler) Descriptor() string {
	return h.handler.Descriptor()
}

func (h stableHashHandler) HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
	return h.handler.HandleTransact(ctx, code, data)
}

func (h stableHashHandler) InterfaceHash() string {
	return h.handler.InterfaceHash()
}

func (h stableHashHandler) StabilityLevel() StabilityLevel {
	return h.level
}

type stableVersionHashHandler struct {
	handler interface {
		Handler
		InterfaceVersionProvider
		InterfaceHashProvider
	}
	level StabilityLevel
}

func (h stableVersionHashHandler) Descriptor() string {
	return h.handler.Descriptor()
}

func (h stableVersionHashHandler) HandleTransact(ctx context.Context, code uint32, data *Parcel) (*Parcel, error) {
	return h.handler.HandleTransact(ctx, code, data)
}

func (h stableVersionHashHandler) InterfaceVersion() int32 {
	return h.handler.InterfaceVersion()
}

func (h stableVersionHashHandler) InterfaceHash() string {
	return h.handler.InterfaceHash()
}

func (h stableVersionHashHandler) StabilityLevel() StabilityLevel {
	return h.level
}
