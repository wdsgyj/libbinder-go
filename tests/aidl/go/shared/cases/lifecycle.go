package cases

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/wdsgyj/libbinder-go/binder"
	shared "github.com/wdsgyj/libbinder-go/tests/aidl/go/shared/generated/com/wdsgyj/libbinder/aidltest/shared"
)

func VerifyLifecycleDiscovery(ctx context.Context, sm binder.ServiceManager, name string, expectPrefix string) error {
	if sm == nil {
		return fmt.Errorf("nil service manager")
	}
	waited, err := sm.WaitService(ctx, name)
	if err != nil {
		return fmt.Errorf("WaitService(%q): %w", name, err)
	}
	if waited == nil {
		return fmt.Errorf("WaitService(%q) returned nil binder", name)
	}

	checked, err := sm.CheckService(ctx, name)
	if err != nil {
		return fmt.Errorf("CheckService(%q): %w", name, err)
	}
	if checked == nil {
		return fmt.Errorf("CheckService(%q) returned nil binder", name)
	}

	services, err := sm.ListServices(ctx, binder.DumpPriorityAll)
	if err != nil {
		return fmt.Errorf("ListServices: %w", err)
	}
	if !slices.Contains(services, name) {
		return fmt.Errorf("ListServices missing %q: %v", name, services)
	}

	svc := shared.NewIBaselineServiceClient(checked)
	if svc == nil {
		return fmt.Errorf("typed baseline client is nil")
	}
	ping, err := svc.Ping(ctx)
	if err != nil {
		return fmt.Errorf("Ping: %w", err)
	}
	if !ping {
		return fmt.Errorf("Ping = false, want true")
	}
	echo, err := svc.EchoNullable(ctx, lifecycleStringPtr("hello"))
	if err != nil {
		return fmt.Errorf("EchoNullable: %w", err)
	}
	wantEcho := expectPrefix + ":hello"
	if echo == nil || *echo != wantEcho {
		return fmt.Errorf("EchoNullable = %#v, want %q", echo, wantEcho)
	}
	return nil
}

func lifecycleStringPtr(v string) *string {
	return &v
}

func WaitForBinderDeathAfter(ctx context.Context, service binder.Binder, delay time.Duration, kill func() error) error {
	if service == nil {
		return fmt.Errorf("nil binder")
	}
	sub, err := service.WatchDeath(ctx)
	if err != nil {
		return fmt.Errorf("WatchDeath: %w", err)
	}
	defer sub.Close()

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		if err := kill(); err != nil {
			return fmt.Errorf("kill: %w", err)
		}
	case <-sub.Done():
		if err := sub.Err(); err != nil {
			return fmt.Errorf("subscription completed early: %w", err)
		}
		return fmt.Errorf("subscription completed early without error")
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-sub.Done():
		if err := sub.Err(); err != binder.ErrDeadObject {
			return fmt.Errorf("WatchDeath.Err = %v, want %v", err, binder.ErrDeadObject)
		}
		return nil
	}
}
