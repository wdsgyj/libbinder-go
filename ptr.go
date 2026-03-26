package libbinder

// Ptr returns a pointer to v.
func Ptr[T any](v T) *T {
	return &v
}

// BoolPtr returns a pointer to v.
func BoolPtr(v bool) *bool {
	return Ptr(v)
}

// StringPtr returns a pointer to v.
func StringPtr(v string) *string {
	return Ptr(v)
}

// IntPtr returns a pointer to v.
func IntPtr(v int) *int {
	return Ptr(v)
}

// Int8Ptr returns a pointer to v.
func Int8Ptr(v int8) *int8 {
	return Ptr(v)
}

// Int16Ptr returns a pointer to v.
func Int16Ptr(v int16) *int16 {
	return Ptr(v)
}

// Int32Ptr returns a pointer to v.
func Int32Ptr(v int32) *int32 {
	return Ptr(v)
}

// Int64Ptr returns a pointer to v.
func Int64Ptr(v int64) *int64 {
	return Ptr(v)
}

// UintPtr returns a pointer to v.
func UintPtr(v uint) *uint {
	return Ptr(v)
}

// Uint8Ptr returns a pointer to v.
func Uint8Ptr(v uint8) *uint8 {
	return Ptr(v)
}

// Uint16Ptr returns a pointer to v.
func Uint16Ptr(v uint16) *uint16 {
	return Ptr(v)
}

// Uint32Ptr returns a pointer to v.
func Uint32Ptr(v uint32) *uint32 {
	return Ptr(v)
}

// Uint64Ptr returns a pointer to v.
func Uint64Ptr(v uint64) *uint64 {
	return Ptr(v)
}

// UintptrPtr returns a pointer to v.
func UintptrPtr(v uintptr) *uintptr {
	return Ptr(v)
}

// BytePtr returns a pointer to v.
func BytePtr(v byte) *byte {
	return Ptr(v)
}

// RunePtr returns a pointer to v.
func RunePtr(v rune) *rune {
	return Ptr(v)
}

// Float32Ptr returns a pointer to v.
func Float32Ptr(v float32) *float32 {
	return Ptr(v)
}

// Float64Ptr returns a pointer to v.
func Float64Ptr(v float64) *float64 {
	return Ptr(v)
}

// Complex64Ptr returns a pointer to v.
func Complex64Ptr(v complex64) *complex64 {
	return Ptr(v)
}

// Complex128Ptr returns a pointer to v.
func Complex128Ptr(v complex128) *complex128 {
	return Ptr(v)
}
