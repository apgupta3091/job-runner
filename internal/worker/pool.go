package worker

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/apgupta3091/job-runner/internal/job"
)

// JobStore interface for updating job status
type JobStore interface {
	Update(job *job.Job)
}

// Pool manages a pool of worker goroutines
type Pool struct {
	workerCount int
	jobQueue    <-chan *job.Job
	store       JobStore
	wg          sync.WaitGroup
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewPool creates a new worker pool with cancellable context
func NewPool(workerCount int, jobQueue <-chan *job.Job, store JobStore) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		workerCount: workerCount,
		jobQueue:    jobQueue,
		store:       store,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start spawns all worker goroutines
func (p *Pool) Start() {
	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
	log.Printf("Started %d workers", p.workerCount)
}

// worker is the main loop for each worker goroutine
func (p *Pool) worker(id int) {
	defer p.wg.Done()
	log.Printf("Worker %d started", id)

	for {
		select {
		case job, ok := <-p.jobQueue:
			if !ok {
				log.Printf("Worker %d: queue closed, exiting", id)
				return
			}
			p.processJob(id, job)
		case <-p.ctx.Done():
			log.Printf("Worker %d: context cancelled, exiting", id)
			return
		}
	}
}

// processJob handles a single job with panic recovery
func (p *Pool) processJob(workerID int, j *job.Job) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Worker %d: panic processing job %s: %v", workerID, j.ID, r)
			j.Status = job.StatusFailed
			j.Error = "panic during processing"
			now := time.Now()
			j.CompletedAt = &now
			p.store.Update(j)
		}
	}()

	log.Printf("Worker %d: processing job %s", workerID, j.ID)

	// Mark as running
	j.Status = job.StatusRunning
	now := time.Now()
	j.StartedAt = &now
	p.store.Update(j)

	// Simulate work
	time.Sleep(2 * time.Second)

	// Mark as completed
	j.Status = job.StatusCompleted
	completed := time.Now()
	j.CompletedAt = &completed
	p.store.Update(j)

	log.Printf("Worker %d: completed job %s", workerID, j.ID)
}

// Shutdown gracefully stops all workers with timeout
func (p *Pool) Shutdown(timeout time.Duration) error {
	log.Println("Shutting down worker pool...")

	p.cancel()

	// Wait for workers with timeout
	done := make(chan struct{})
	go func() {
		p.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All workers stopped gracefully")
		return nil
	case <-time.After(timeout):
		return errors.New("worker shutdown timeout")
	}
}
