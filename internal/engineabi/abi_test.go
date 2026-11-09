package engineabi

import (
	"context"
	"testing"
	"time"
)

func Test_should_retry_cancel_when_request_registration_is_late(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	finished := make(chan struct{})
	defer close(finished)

	attempts := make(chan int, 1)
	go func() {
		calls := 0
		cancelWhenRequested(ctx, finished, func() bool {
			calls++
			return calls == 2
		})
		attempts <- calls
	}()

	select {
	case calls := <-attempts:
		if calls != 2 {
			t.Fatalf("expected retry after unregistered request, got %d attempts", calls)
		}
	case <-time.After(time.Second):
		t.Fatal("cancellation was not retried")
	}
}
