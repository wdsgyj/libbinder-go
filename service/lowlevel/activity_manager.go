package lowlevel

import (
	"context"
	"sync"

	api "github.com/wdsgyj/libbinder-go/binder"
)

const (
	ActivityServiceName     = "activity"
	ActivityTaskServiceName = "activity_task"
)

func LookupActivityService(ctx context.Context, sm api.ServiceManager) (api.Binder, error) {
	return lookupNamedService(ctx, sm, ActivityServiceName)
}

func WaitActivityService(ctx context.Context, sm api.ServiceManager) (api.Binder, error) {
	return waitNamedService(ctx, sm, ActivityServiceName)
}

func LookupActivityTaskService(ctx context.Context, sm api.ServiceManager) (api.Binder, error) {
	return lookupNamedService(ctx, sm, ActivityTaskServiceName)
}

func WaitActivityTaskService(ctx context.Context, sm api.ServiceManager) (api.Binder, error) {
	return waitNamedService(ctx, sm, ActivityTaskServiceName)
}

func LookupActivityManager(ctx context.Context, sm api.ServiceManager) (*ActivityManager, error) {
	activity, err := LookupActivityService(ctx, sm)
	if err != nil {
		return nil, err
	}
	activityTask, err := LookupActivityTaskService(ctx, sm)
	if err != nil {
		return nil, err
	}
	return NewActivityManager(activity, activityTask), nil
}

func WaitActivityManager(ctx context.Context, sm api.ServiceManager) (*ActivityManager, error) {
	activity, err := WaitActivityService(ctx, sm)
	if err != nil {
		return nil, err
	}
	activityTask, err := WaitActivityTaskService(ctx, sm)
	if err != nil {
		return nil, err
	}
	return NewActivityManager(activity, activityTask), nil
}

type ActivityManager struct {
	activity     api.Binder
	activityTask api.Binder
}

func NewActivityManager(activity api.Binder, activityTask api.Binder) *ActivityManager {
	return &ActivityManager{
		activity:     activity,
		activityTask: activityTask,
	}
}

// GetService mirrors android.app.ActivityManager.getService().
func (m *ActivityManager) GetService() api.Binder {
	if m == nil {
		return nil
	}
	return m.activity
}

// GetTaskService mirrors android.app.ActivityManager.getTaskService().
func (m *ActivityManager) GetTaskService() api.Binder {
	if m == nil {
		return nil
	}
	return m.activityTask
}

func (m *ActivityManager) ActivityService() api.Binder {
	return m.GetService()
}

func (m *ActivityManager) ActivityTaskService() api.Binder {
	return m.GetTaskService()
}

// ActivityManagerProvider mirrors the Java-side Singleton pattern and lazily
// resolves both Binder services once.
type ActivityManagerProvider struct {
	sm api.ServiceManager

	mu     sync.Mutex
	cached *ActivityManager
}

func NewActivityManagerProvider(sm api.ServiceManager) *ActivityManagerProvider {
	return &ActivityManagerProvider{sm: sm}
}

func (p *ActivityManagerProvider) Get(ctx context.Context) (*ActivityManager, error) {
	if p == nil || p.sm == nil {
		return nil, api.ErrUnsupported
	}

	p.mu.Lock()
	if p.cached != nil {
		cached := p.cached
		p.mu.Unlock()
		return cached, nil
	}
	p.mu.Unlock()

	manager, err := LookupActivityManager(ctx, p.sm)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	if p.cached == nil {
		p.cached = manager
	} else {
		manager = p.cached
	}
	p.mu.Unlock()
	return manager, nil
}

func (p *ActivityManagerProvider) Invalidate() {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.cached = nil
	p.mu.Unlock()
}

func lookupNamedService(ctx context.Context, sm api.ServiceManager, name string) (api.Binder, error) {
	if sm == nil {
		return nil, api.ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}
	service, err := sm.CheckService(ctx, name)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, api.ErrNoService
	}
	return service, nil
}

func waitNamedService(ctx context.Context, sm api.ServiceManager, name string) (api.Binder, error) {
	if sm == nil {
		return nil, api.ErrUnsupported
	}
	if ctx == nil {
		ctx = context.Background()
	}
	service, err := sm.WaitService(ctx, name)
	if err != nil {
		return nil, err
	}
	if service == nil {
		return nil, api.ErrNoService
	}
	return service, nil
}
