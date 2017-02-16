package stats

import (
	"encoding/json"
	"sync"
	"time"

	"golang.org/x/net/context"
)

type Repository interface {
	Get(ctx context.Context, id string) Stats
}

type Collector interface {
	Repository
	Dump() []byte
	Close()
}

type Statistics struct {
	TS          time.Time
	CPUUserTime uint64
	CPUSysTime  uint64
	MemoryRSS   uint64
	MemoryVS    uint64
}

type Stats interface {
	Update(data Statistics)
}

type collector struct {
	sync.RWMutex
	workers map[string]*stats
}

func New() Collector {
	c := collector{
		workers: make(map[string]*stats),
	}
	return &c
}

func (c *collector) Get(ctx context.Context, id string) Stats {
	c.RLock()
	s, ok := c.workers[id]
	c.RUnlock()
	if ok {
		return s
	}

	c.Lock()
	defer c.Unlock()
	s, ok = c.workers[id]
	if ok {
		return s
	}

	s = &stats{}
	c.workers[id] = s
	return s
}

func (c *collector) Dump() []byte {
	c.RLock()
	defer c.RUnlock()
	body, _ := json.Marshal(c.workers)
	return body
}

func (*collector) Close() {}

type stats struct {
	sync.Mutex
	Statistics
}

func (s *stats) Update(data Statistics) {
	s.Lock()
	defer s.Unlock()
	s.CPUUserTime = data.CPUUserTime
	s.CPUSysTime = data.CPUSysTime
	s.MemoryRSS = data.MemoryRSS
	s.MemoryVS = data.MemoryVS
	s.TS = data.TS
}

func (s *stats) MarshalJSON() ([]byte, error) {
	s.Lock()
	defer s.Unlock()
	return json.Marshal(s.Statistics)
}
