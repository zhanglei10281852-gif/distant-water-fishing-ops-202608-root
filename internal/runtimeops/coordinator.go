package runtimeops

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Step func(context.Context) error

type Pipeline struct {
	mu        sync.Mutex
	completed map[string]bool
}

func NewPipeline() *Pipeline { return &Pipeline{completed: map[string]bool{}} }

func (p *Pipeline) Run(ctx context.Context, steps map[string]Step, order []string) error {
	for _, name := range order {
		p.mu.Lock()
		done := p.completed[name]
		p.mu.Unlock()
		if done {
			continue
		}
		step := steps[name]
		if step == nil {
			return fmt.Errorf("step %s: %w", name, ErrNotFound)
		}
		if err := step(ctx); err != nil {
			return fmt.Errorf("step %s: %w", name, err)
		}
		p.mu.Lock()
		p.completed[name] = true
		p.mu.Unlock()
	}
	return nil
}

func (p *Pipeline) Snapshot() map[string]bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make(map[string]bool, len(p.completed))
	for key, value := range p.completed {
		out[key] = value
	}
	return out
}

func (p *Pipeline) Restore(snapshot map[string]bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.completed = snapshot
}

func Retry(ctx context.Context, attempts int, backoff time.Duration, operation func(context.Context) error) error {
	if attempts < 1 {
		return ErrConflict
	}
	var last error
	for attempt := 0; attempt < attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := operation(ctx); err == nil {
			return nil
		} else {
			last = err
		}
		if attempt == attempts-1 {
			break
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return ctx.Err()
		case <-timer.C:
		}
	}
	return last
}

type Processor struct {
	Store *Store
	Clock func() time.Time
	Lease time.Duration
}

func (p Processor) RunOnce(ctx context.Context, tenant string, handler func(context.Context, Job) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	jobs, err := p.Store.ClaimJobs(ctx, tenant, p.Clock(), 1, p.Lease)
	if err != nil {
		return err
	}
	for _, job := range jobs {
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = handler(ctx, job); err != nil {
			if failErr := p.Store.FailJob(ctx, tenant, job.ID, job.Attempts, p.Clock()); failErr != nil {
				return errors.Join(err, failErr)
			}
			return err
		}
		if err = p.Store.CompleteJob(ctx, tenant, job.ID, job.Attempts); err != nil {
			return err
		}
	}
	return nil
}
