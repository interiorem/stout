package semaphore

import (
	"testing"
	"time"

	"golang.org/x/net/context"
)

func TestSema(t *testing.T) {
	sleep := time.Millisecond * 100
	ctx := context.Background()
	sm := New(1)
	sm.Acquire(ctx)
	ctx, _ = context.WithTimeout(context.Background(), sleep)
	u := time.Now()
	err := sm.Acquire(ctx)
	d := time.Now().Sub(u)
	if d < sleep {
		t.Fail()
	}
	if !(err == context.DeadlineExceeded || err == context.Canceled) {
		t.Fatalf("err must be either %v or %v, not %v", context.DeadlineExceeded, context.Canceled, err)
	}
	sm.Release()
}
