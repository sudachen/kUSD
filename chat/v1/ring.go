package v1

import (
	"sync"
	"time"
	"github.com/kowala-tech/kUSD/log"
)

type ring struct {
	queue chan *message
	mu         sync.Mutex
	known      map[Hash]int64
	bf         [messageRingLength]*message
	head, tail uint64
}

func newRing() *ring {
	return &ring{
		known: make(map[Hash]int64),
		queue:make(chan *message, messageQueueLimit),
	}
}

func (r *ring) expire(quit chan struct{}) {
	clock := time.NewTicker(expireTimeout)
	for {
		if done2(quit, clock.C) {
			return
		}
		r.mu.Lock()
		for k, d := range r.known {
			if d > time.Now().Unix() {
				delete(r.known, k)
			}
		}
		r.mu.Unlock()
	}
}

func (r *ring) dequeue(quit chan struct{}) {
	for {
		select {
		case <-quit:
			return
		case m := <-r.queue:
			r.put(m)
		}
	}
}

func (r *ring) enqueue(m *message) {
	r.queue <- m
}

func (r *ring) put(m *message) {
	hash := m.hash()    // can take a time
	dt := m.deathTime() // can take a time

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.known[hash]; ok {
		return
	}

	index := r.tail % messageRingLength
	if r.tail == r.head+messageRingLength {
		r.head += 1
	}

	log.Trace("put to ring", "hash", m.hash(), "index", r.tail, "hash")

	r.bf[index] = m
	r.known[hash] = dt
	r.tail += 1
}

func (r *ring) get(oldIndex uint64) (newIndex uint64, m *message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	newIndex = oldIndex
	if r.head > newIndex {
		newIndex = r.head
	}
	if newIndex < r.tail {
		m = r.bf[newIndex%messageRingLength]
		newIndex += 1
	}
	return
}
