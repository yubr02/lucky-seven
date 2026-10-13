package realtime

import (
	"sync"
	"testing"
)

func TestNotificationsCoalesceWithoutBlocking(t *testing.T) {
	h := New()
	defer h.Close()
	ch := make(chan struct{}, 1)
	h.clients[ch] = struct{}{}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				h.Publish()
			}
		}()
	}
	wg.Wait()
	if len(ch) != 1 {
		t.Fatalf("queued %d notifications", len(ch))
	}
}
