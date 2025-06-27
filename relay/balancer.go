package relay

import "sync/atomic"

type RoundRobinBalancer struct {
	counter uint64
	targets []string
}

func NewRoundRobinBalancer(targets []string) *RoundRobinBalancer {
	return &RoundRobinBalancer{targets: targets}
}

func (rr *RoundRobinBalancer) Next() string {
	if len(rr.targets) == 0 {
		return ""
	}
	idx := atomic.AddUint64(&rr.counter, 1)
	return rr.targets[int(idx-1)%len(rr.targets)]
}
