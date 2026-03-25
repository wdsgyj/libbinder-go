package libbinder

import (
	"sync"

	api "github.com/wdsgyj/libbinder-go/binder"
)

type trackedSubscription struct {
	inner api.Subscription

	releaseOnce sync.Once
	releaseFn   func() error
}

func newTrackedSubscription(inner api.Subscription, releaseFn func() error) api.Subscription {
	sub := &trackedSubscription{
		inner:     inner,
		releaseFn: releaseFn,
	}

	go func() {
		if sub.inner == nil {
			return
		}
		<-sub.inner.Done()
		sub.release()
	}()

	return sub
}

func (s *trackedSubscription) Done() <-chan struct{} {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.Done()
}

func (s *trackedSubscription) Err() error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.Err()
}

func (s *trackedSubscription) Close() error {
	if s == nil || s.inner == nil {
		return nil
	}
	return s.inner.Close()
}

func (s *trackedSubscription) release() error {
	if s == nil || s.releaseFn == nil {
		return nil
	}

	var err error
	s.releaseOnce.Do(func() {
		err = s.releaseFn()
	})
	return err
}
