package runtime

import "testing"

func TestRefTrackerAcquireAndRelease(t *testing.T) {
	tracker := NewRefTracker()
	tracker.RetainBinder(7)

	need, wait := tracker.BeginAcquire(7)
	if !need {
		t.Fatalf("BeginAcquire need = %v, want true", need)
	}
	if wait != nil {
		t.Fatalf("BeginAcquire wait = %v, want nil", wait)
	}

	if shouldRelease := tracker.FinishAcquire(7, true); shouldRelease {
		t.Fatal("FinishAcquire should not request release while binder ref is held")
	}

	if shouldRelease := tracker.ReleaseBinder(7); !shouldRelease {
		t.Fatal("ReleaseBinder should request kernel release for last acquired ref")
	}
}

func TestRefTrackerWaitsForInFlightAcquire(t *testing.T) {
	tracker := NewRefTracker()
	tracker.RetainBinder(9)

	need, wait := tracker.BeginAcquire(9)
	if !need || wait != nil {
		t.Fatalf("first BeginAcquire = (%v, %v), want (true, nil)", need, wait)
	}

	need, wait = tracker.BeginAcquire(9)
	if need {
		t.Fatal("second BeginAcquire should not start a parallel acquire")
	}
	if wait == nil {
		t.Fatal("second BeginAcquire should return a wait channel")
	}

	if shouldRelease := tracker.FinishAcquire(9, true); shouldRelease {
		t.Fatal("FinishAcquire should not request release while binder ref is held")
	}

	select {
	case <-wait:
	default:
		t.Fatal("wait channel should be closed after FinishAcquire")
	}
}

func TestRefTrackerReleaseAfterAcquireCompletes(t *testing.T) {
	tracker := NewRefTracker()
	tracker.RetainBinder(11)

	need, _ := tracker.BeginAcquire(11)
	if !need {
		t.Fatal("BeginAcquire should need an acquire")
	}

	if shouldRelease := tracker.ReleaseBinder(11); shouldRelease {
		t.Fatal("ReleaseBinder should wait for in-flight acquire before releasing")
	}

	if shouldRelease := tracker.FinishAcquire(11, true); !shouldRelease {
		t.Fatal("FinishAcquire should request release when last ref vanished during acquire")
	}
}

func TestRefTrackerWatchRefsPinHandle(t *testing.T) {
	tracker := NewRefTracker()
	tracker.RetainBinder(13)
	tracker.RetainWatch(13)

	need, _ := tracker.BeginAcquire(13)
	if !need {
		t.Fatal("BeginAcquire should need an acquire")
	}
	if shouldRelease := tracker.FinishAcquire(13, true); shouldRelease {
		t.Fatal("FinishAcquire should not request release while refs are held")
	}

	if shouldRelease := tracker.ReleaseBinder(13); shouldRelease {
		t.Fatal("ReleaseBinder should not release while a watch ref is still held")
	}
	if shouldRelease := tracker.ReleaseWatch(13); !shouldRelease {
		t.Fatal("ReleaseWatch should release when the last ref disappears")
	}
}
