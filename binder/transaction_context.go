package binder

import "context"

// TransactionContext exposes runtime metadata about the current Binder call.
type TransactionContext struct {
	Code       uint32
	Flags      Flags
	CallingPID int32
	CallingUID uint32
	Local      bool
}

type transactionContextKey struct{}

// WithTransactionContext attaches Binder transaction metadata to ctx.
//
// This is intended for runtime and generated-code plumbing.
func WithTransactionContext(ctx context.Context, tx TransactionContext) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, transactionContextKey{}, tx)
}

// TransactionContextFromContext returns Binder transaction metadata from ctx.
func TransactionContextFromContext(ctx context.Context) (TransactionContext, bool) {
	if ctx == nil {
		return TransactionContext{}, false
	}
	tx, ok := ctx.Value(transactionContextKey{}).(TransactionContext)
	return tx, ok
}

func CallingPID(ctx context.Context) (int32, bool) {
	tx, ok := TransactionContextFromContext(ctx)
	if !ok {
		return 0, false
	}
	return tx.CallingPID, true
}

func CallingUID(ctx context.Context) (uint32, bool) {
	tx, ok := TransactionContextFromContext(ctx)
	if !ok {
		return 0, false
	}
	return tx.CallingUID, true
}
