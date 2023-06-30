package endpoints

import (
	"sync"
	"time"
)

type Endpoint struct {
	Server   string
	RespTime time.Duration
}

type ActiveEndpoints struct {
	mu   sync.RWMutex
	list []Endpoint
}

func (a *ActiveEndpoints) ReadEndpoints() []Endpoint {
	a.mu.RLock()
	l := a.list
	a.mu.RUnlock()
	return l
}

func (a *ActiveEndpoints) WriteEndpoints(l []Endpoint) {
	a.mu.Lock()
	a.list = l
	a.mu.Unlock()
}
