package main

import (
	"context"
	"sync"
	"time"

	"opdeals/internal/clearance"
	"opdeals/internal/logx"
)

// freshFor is how long a just-solved cookie is assumed good. Concurrent callers
// arriving inside this window reuse that solve instead of driving the browser
// again — a scrape cycle blocks on several hosts at once when it starts cold.
const freshFor = 60 * time.Second

type refresher struct {
	store   *clearance.Store
	hosts   []string
	budget  time.Duration
	headles bool
	visible bool
	log     *logx.Logger

	mu       sync.Mutex
	solvedAt map[string]time.Time
}

func newRefresher(store *clearance.Store, hosts []string, budget time.Duration, headless, visible bool, log *logx.Logger) *refresher {
	return &refresher{
		store:    store,
		hosts:    hosts,
		budget:   budget,
		headles:  headless,
		visible:  visible,
		log:      log,
		solvedAt: map[string]time.Time{},
	}
}

// one re-earns a single host's clearance, coalescing callers that arrive while
// a solve is in flight or just finished.
func (r *refresher) one(ctx context.Context, solveProxy, host string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if at, ok := r.solvedAt[host]; ok && time.Since(at) < freshFor {
		return nil
	}
	if err := r.run(ctx, []string{host}, solveProxy); err != nil {
		return err
	}
	r.solvedAt[host] = time.Now()
	return nil
}

// all re-earns every host's clearance.
func (r *refresher) all(ctx context.Context, solveProxy string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.run(ctx, r.hosts, solveProxy); err != nil {
		return err
	}
	now := time.Now()
	for _, h := range r.hosts {
		r.solvedAt[h] = now
	}
	return nil
}

func (r *refresher) run(ctx context.Context, hosts []string, solveProxy string) error {
	solveCtx, cancel := context.WithTimeout(ctx, time.Duration(len(hosts))*r.budget+60*time.Second)
	defer cancel()

	entries, failed := solveAll(solveCtx, hosts, r.budget, solveProxy, r.headles, r.visible, r.log)
	if len(entries) == 0 {
		return errNoneSolved
	}
	if err := merge(r.store, entries, r.hosts); err != nil {
		return err
	}
	r.log.Printf("wrote %d/%d host(s) to %s", len(entries), len(hosts), r.store.Path())
	if len(failed) > 0 {
		return unsolvedError(failed)
	}
	return nil
}
