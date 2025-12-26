package metrics

import (
	"fmt"
	"sync"
)

var (
	mu         sync.RWMutex
	collectors = make(map[string]Collector)
)

func Register(c Collector) error {
	mu.Lock()
	defer mu.Unlock()

	id := c.ID()
	if id == "" {
		return fmt.Errorf("collector ID cannot be empty")
	}
	if _, exists := collectors[id]; exists {
		return fmt.Errorf("collector already registered: %s", id)
	}
	collectors[id] = c
	return nil
}

func All() []Collector {
	mu.RLock()
	defer mu.RUnlock()

	out := make([]Collector, 0, len(collectors))
	for _, c := range collectors {
		out = append(out, c)
	}
	return out
}
