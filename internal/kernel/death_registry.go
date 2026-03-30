package kernel

import (
	"context"
	"sync"
	"sync/atomic"

	api "github.com/wdsgyj/libbinder-go/binder"
)

type deathRegistry struct {
	request func(context.Context, uint32, uintptr) error
	clear   func(context.Context, uint32, uintptr) error

	mu         sync.Mutex
	nextCookie uint64
	closed     bool
	byHandle   map[uint32]*deathWatch
	byCookie   map[uintptr]*deathWatch
}

type deathWatch struct {
	handle uint32
	cookie uintptr

	subs        map[*deathSubscription]struct{}
	closingSubs map[*deathSubscription]struct{}
}

type deathSubscription struct {
	registry *deathRegistry
	watch    *deathWatch
	done     chan struct{}

	mu   sync.Mutex
	err  error
	once sync.Once
}

func newDeathRegistry(
	request func(context.Context, uint32, uintptr) error,
	clear func(context.Context, uint32, uintptr) error,
) *deathRegistry {
	return &deathRegistry{
		request:  request,
		clear:    clear,
		byHandle: make(map[uint32]*deathWatch),
		byCookie: make(map[uintptr]*deathWatch),
	}
}

func (r *deathRegistry) Watch(ctx context.Context, handle uint32) (api.Subscription, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	sub, needsRequest, err := r.attach(handle)
	if err != nil {
		return nil, err
	}

	if needsRequest {
		if err := r.request(ctx, handle, sub.watch.cookie); err != nil {
			r.rollback(sub.watch)
			return nil, err
		}
	}

	r.bridgeContext(ctx, sub)
	return sub, nil
}

func (r *deathRegistry) attach(handle uint32) (*deathSubscription, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.closed {
		return nil, false, ErrDriverClosed
	}

	if watch := r.byHandle[handle]; watch != nil {
		sub := newDeathSubscription(r, watch)
		watch.subs[sub] = struct{}{}
		return sub, false, nil
	}

	cookie := uintptr(atomic.AddUint64(&r.nextCookie, 1))
	watch := &deathWatch{
		handle:      handle,
		cookie:      cookie,
		subs:        make(map[*deathSubscription]struct{}),
		closingSubs: make(map[*deathSubscription]struct{}),
	}
	sub := newDeathSubscription(r, watch)
	watch.subs[sub] = struct{}{}
	r.byHandle[handle] = watch
	r.byCookie[cookie] = watch
	return sub, true, nil
}

func (r *deathRegistry) rollback(watch *deathWatch) {
	if watch == nil {
		return
	}

	r.mu.Lock()
	if current := r.byHandle[watch.handle]; current == watch {
		delete(r.byHandle, watch.handle)
	}
	if current := r.byCookie[watch.cookie]; current == watch {
		delete(r.byCookie, watch.cookie)
	}
	r.mu.Unlock()
}

func (r *deathRegistry) bridgeContext(ctx context.Context, sub *deathSubscription) {
	if sub == nil {
		return
	}
	done := ctx.Done()
	if done == nil {
		return
	}

	go func() {
		select {
		case <-done:
			_ = sub.Close()
		case <-sub.Done():
		}
	}()
}

func (r *deathRegistry) removeSub(sub *deathSubscription) error {
	if sub == nil {
		return nil
	}

	var (
		clearHandle uint32
		clearCookie uintptr
		needsClear  bool
	)

	r.mu.Lock()
	watch := sub.watch
	if watch == nil {
		r.mu.Unlock()
		sub.finish(nil)
		return nil
	}
	if current := r.byHandle[watch.handle]; current != watch {
		r.mu.Unlock()
		sub.finish(nil)
		return nil
	}

	delete(watch.subs, sub)
	if len(watch.subs) != 0 {
		r.mu.Unlock()
		sub.finish(nil)
		return nil
	}

	if current := r.byHandle[watch.handle]; current == watch {
		delete(r.byHandle, watch.handle)
	}
	watch.closingSubs[sub] = struct{}{}
	clearHandle = watch.handle
	clearCookie = watch.cookie
	needsClear = true
	r.mu.Unlock()

	if !needsClear || r.clear == nil {
		sub.finish(nil)
		return nil
	}

	if err := r.clear(context.Background(), clearHandle, clearCookie); err != nil {
		r.finishClearing(clearCookie, nil)
		return err
	}
	return nil
}

func (r *deathRegistry) NotifyDead(cookie uintptr) {
	r.mu.Lock()
	watch := r.byCookie[cookie]
	if watch == nil {
		r.mu.Unlock()
		return
	}

	delete(r.byCookie, cookie)
	if current := r.byHandle[watch.handle]; current == watch {
		delete(r.byHandle, watch.handle)
	}

	subs := make([]*deathSubscription, 0, len(watch.subs)+len(watch.closingSubs))
	for sub := range watch.subs {
		subs = append(subs, sub)
	}
	for sub := range watch.closingSubs {
		subs = append(subs, sub)
	}
	watch.subs = nil
	watch.closingSubs = nil
	r.mu.Unlock()

	for _, sub := range subs {
		sub.finish(api.ErrDeadObject)
	}
}

func (r *deathRegistry) NotifyCleared(cookie uintptr) {
	r.finishClearing(cookie, nil)
}

func (r *deathRegistry) Close() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true

	subs := make([]*deathSubscription, 0)
	for _, watch := range r.byHandle {
		for sub := range watch.subs {
			subs = append(subs, sub)
		}
		for sub := range watch.closingSubs {
			subs = append(subs, sub)
		}
	}
	r.byHandle = nil
	r.byCookie = nil
	r.mu.Unlock()

	for _, sub := range subs {
		sub.finish(nil)
	}
}

func newDeathSubscription(registry *deathRegistry, watch *deathWatch) *deathSubscription {
	return &deathSubscription{
		registry: registry,
		watch:    watch,
		done:     make(chan struct{}),
	}
}

func (s *deathSubscription) Done() <-chan struct{} {
	if s == nil {
		return nil
	}
	return s.done
}

func (s *deathSubscription) Err() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

func (s *deathSubscription) Close() error {
	if s == nil || s.registry == nil {
		return nil
	}
	return s.registry.removeSub(s)
}

func (s *deathSubscription) finish(err error) {
	if s == nil {
		return
	}

	s.once.Do(func() {
		s.mu.Lock()
		s.err = err
		close(s.done)
		s.mu.Unlock()
	})
}

func (r *deathRegistry) finishClearing(cookie uintptr, err error) {
	r.mu.Lock()
	watch := r.byCookie[cookie]
	if watch == nil {
		r.mu.Unlock()
		return
	}

	delete(r.byCookie, cookie)
	if current := r.byHandle[watch.handle]; current == watch {
		delete(r.byHandle, watch.handle)
	}

	subs := make([]*deathSubscription, 0, len(watch.closingSubs))
	for sub := range watch.closingSubs {
		subs = append(subs, sub)
	}
	watch.closingSubs = nil
	r.mu.Unlock()

	for _, sub := range subs {
		sub.finish(err)
	}
}
