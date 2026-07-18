package lifecycle

import (
	"sync"

	"github.com/OlegGitH/epistemic-engine/apps/control-plane/internal/domain"
)

// Bus is a process-local lifecycle fan-out. Durable state remains in the
// repository; clients reconnect by reading a graph snapshot before subscribing.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan domain.LifecycleEvent]struct{}
}

func NewBus() *Bus {
	return &Bus{subscribers: map[string]map[chan domain.LifecycleEvent]struct{}{}}
}

func (b *Bus) Publish(event domain.LifecycleEvent) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	for subscriber := range b.subscribers[event.RunID] {
		select {
		case subscriber <- event:
		default:
			// A slow UI must not block ingestion or verification.
		}
	}
}

func (b *Bus) Subscribe(runID string) (<-chan domain.LifecycleEvent, func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	channel := make(chan domain.LifecycleEvent, 16)
	if b.subscribers[runID] == nil {
		b.subscribers[runID] = map[chan domain.LifecycleEvent]struct{}{}
	}
	b.subscribers[runID][channel] = struct{}{}
	return channel, func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.subscribers[runID], channel)
		close(channel)
	}
}
