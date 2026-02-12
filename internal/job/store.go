package job

import (
	"context"
	"errors"
	"sync"
)

// Store manages jobs in memory with a bounded queue
type Store struct {
	mu    sync.RWMutex
	jobs  map[string]*Job
	queue chan *Job
}

// NewStore creates a new in-memory store with a bounded queue
func NewStore(queueSize int) *Store {
	return &Store{
		jobs:  make(map[string]*Job),
		queue: make(chan *Job, queueSize),
	}
}

// Create adds a job to the store and queue
func (s *Store) Create(ctx context.Context, job *Job) error {
	s.mu.Lock()
	s.jobs[job.ID] = job
	s.mu.Unlock()

	// Try to add to queue
	select {
	case s.queue <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return errors.New("queue full")
	}
}

// Get retrieves a job by ID
func (s *Store) Get(id string) (*Job, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.jobs[id]
	if !ok {
		return nil, errors.New("job not found")
	}
	return job, nil
}

// Update modifies an existing job (called by workers)
func (s *Store) Update(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
}

// Queue returns the job queue channel (for workers to consume)
func (s *Store) Queue() <-chan *Job {
	return s.queue
}

// Close closes the queue channel (no more jobs accepted)
func (s *Store) Close() {
	close(s.queue)
}
