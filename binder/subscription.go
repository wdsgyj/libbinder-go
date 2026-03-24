package binder

// Subscription represents a cancellable runtime subscription, such as death notification.
type Subscription interface {
	Done() <-chan struct{}
	Err() error
	Close() error
}
