package leaderboard

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Broadcaster runs a single background goroutine that polls the leaderboard
// on a fixed interval and fans the encoded JSON out to every subscribed SSE
// client channel.
//
// Without this, N connected clients each issue their own Redis+Postgres
// query on every tick → O(N) reads per interval. With it, cost is O(active
// ranges) regardless of client count — typically just 1–3 reads total.
type Broadcaster struct {
	svc          *Service
	pollInterval time.Duration

	mu          sync.Mutex
	subscribers map[string]map[chan []byte]struct{} // rangeName → set of client chans
}

// NewBroadcaster creates a Broadcaster backed by the given service.
// interval is how often Redis is polled (3s is a good default).
func NewBroadcaster(svc *Service, interval time.Duration) *Broadcaster {
	return &Broadcaster{
		svc:          svc,
		pollInterval: interval,
		subscribers:  make(map[string]map[chan []byte]struct{}),
	}
}

// Subscribe registers a buffered channel that will receive leaderboard JSON
// for the given range on every poll tick. The caller must always call
// Unsubscribe (typically in a defer) to avoid goroutine leaks.
func (b *Broadcaster) Subscribe(rangeName string) chan []byte {
	// Buffer of 1 so a slow client can't block the broadcast loop.
	ch := make(chan []byte, 1)
	b.mu.Lock()
	if b.subscribers[rangeName] == nil {
		b.subscribers[rangeName] = make(map[chan []byte]struct{})
	}
	b.subscribers[rangeName][ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel from the subscriber set and closes it,
// which unblocks the range-for loop in StreamLeaderboard.
func (b *Broadcaster) Unsubscribe(rangeName string, ch chan []byte) {
	b.mu.Lock()
	delete(b.subscribers[rangeName], ch)
	b.mu.Unlock()
	close(ch)
}

// Run starts the poll-and-broadcast loop. It blocks until ctx is cancelled;
// call it in a dedicated goroutine from main.go. The context should be the
// server's root context so all in-flight work is cancelled on shutdown.
func (b *Broadcaster) Run(ctx context.Context) {
	ticker := time.NewTicker(b.pollInterval)
	defer ticker.Stop()

	// Send an immediate snapshot so the first subscriber doesn't stare at
	// a blank screen for up to pollInterval before seeing data.
	b.broadcast(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.broadcast(ctx)
		}
	}
}

// broadcast fetches the leaderboard once per active range and fans out to
// every subscriber. Skipping empty ranges avoids pointless Redis queries when
// nobody is watching.
func (b *Broadcaster) broadcast(ctx context.Context) {
	// Snapshot the set of ranges that currently have subscribers so we can
	// release the lock before doing I/O.
	b.mu.Lock()
	activeRanges := make([]string, 0, len(b.subscribers))
	for r, subs := range b.subscribers {
		if len(subs) > 0 {
			activeRanges = append(activeRanges, r)
		}
	}
	b.mu.Unlock()

	for _, rangeName := range activeRanges {
		entries, err := b.svc.GetLeaderboard(ctx, rangeName)
		if err != nil {
			continue // transient error; skip this tick, don't kill the loop
		}
		data, err := json.Marshal(entries)
		if err != nil {
			continue
		}

		b.mu.Lock()
		for ch := range b.subscribers[rangeName] {
			select {
			case ch <- data:
			default:
				// Client hasn't drained its previous message yet; drop this
				// tick for it rather than blocking the entire broadcast loop.
			}
		}
		b.mu.Unlock()
	}
}
