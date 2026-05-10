package worker

import (
	"context"
	"runtime"
	"sync"

	"golang.org/x/sync/semaphore"
)

type Pool struct {
	workers   int
	sem       *semaphore.Weighted
	tasks     chan func(context.Context) error
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.Mutex
	active    int
	completed int
	failed    int
}

type Config struct {
	Workers    int
	BufferSize int
}

func New(cfg Config) *Pool {
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.GOMAXPROCS(0) * 2
	}
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = cfg.Workers * 2
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		workers:   cfg.Workers,
		sem:       semaphore.NewWeighted(int64(cfg.Workers)),
		tasks:     make(chan func(context.Context) error, cfg.BufferSize),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (p *Pool) Start() {
	for i := 0; i < p.workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
}

func (p *Pool) worker() {
	defer p.wg.Done()
	for {
		select {
		case <-p.ctx.Done():
			return
		case task, ok := <-p.tasks:
			if !ok {
				return
			}
			if err := task(p.ctx); err != nil {
				p.mu.Lock()
				p.failed++
				p.mu.Unlock()
			} else {
				p.mu.Lock()
				p.completed++
				p.mu.Unlock()
			}
			p.sem.Release(1)
		}
	}
}

func (p *Pool) Submit(task func(context.Context) error) bool {
	if p.ctx.Err() != nil {
		return false
	}
	if !p.sem.TryAcquire(1) {
		return false
	}
	select {
	case p.tasks <- task:
		p.mu.Lock()
		p.active++
		p.mu.Unlock()
		return true
	default:
		p.sem.Release(1)
		return false
	}
}

func (p *Pool) SubmitBlocking(ctx context.Context, task func(context.Context) error) error {
	if err := p.sem.Acquire(ctx, 1); err != nil {
		return err
	}
	p.mu.Lock()
	p.active++
	p.mu.Unlock()

	select {
	case p.tasks <- task:
		return nil
	case <-ctx.Done():
		p.sem.Release(1)
		return ctx.Err()
	}
}

func (p *Pool) Stats() (active, completed, failed int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.active, p.completed, p.failed
}

func (p *Pool) Shutdown() {
	p.cancel()
	close(p.tasks)
	p.wg.Wait()
}

func (p *Pool) ShutdownBlocking(ctx context.Context) error {
	p.cancel()
	close(p.tasks)

	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *Pool) Workers() int {
	return p.workers
}