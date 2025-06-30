package relay

import "sync/atomic"

type RoundRobinBalancer struct {
	counter uint64
	targets []string
}

func NewRoundRobinBalancer(targets []string) *RoundRobinBalancer {
	if targets == nil {
		targets = []string{}
	}
	return &RoundRobinBalancer{targets: targets}
}

func (rr *RoundRobinBalancer) Next() string {
	if len(rr.targets) == 0 {
		return ""
	}
	for {
		current := atomic.LoadUint64(&rr.counter)
		next := current + 1
		if next == ^uint64(0) { // 到达最大值，重置为0
			next = 0
		}
		if atomic.CompareAndSwapUint64(&rr.counter, current, next) {
			return rr.targets[int(current)%len(rr.targets)]
		}
	}
}