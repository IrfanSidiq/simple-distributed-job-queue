package inmemrepo

import (
	"context"
	"errors"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"sync"
)

type jobRepository struct {
	mu      	sync.RWMutex
	inMemDb 	map[string]*entity.Job
	tokenIndex 	map[string]*entity.Job
}

func (t *jobRepository) FindByToken(ctx context.Context, key string) (*entity.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	job, exists := t.tokenIndex[key]
	if !exists {
		return nil, errors.New("job with that token not found")
	}
	return job, nil
}

// Save Job
func (t *jobRepository) Save(ctx context.Context, job *entity.Job) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.inMemDb[job.ID] = job

	if job.Token != nil && *job.Token != "" {
		t.tokenIndex[*job.Token] = job
	}
	return nil
}

// Find Job By ID
func (t *jobRepository) FindByID(ctx context.Context, id string) (*entity.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	job, exists := t.inMemDb[id]
	if !exists {
		return nil, errors.New("job not found")
	}
	return job, nil
}

// FindAll Job
func (t *jobRepository) FindAll(ctx context.Context) ([]*entity.Job, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var jobs []*entity.Job
	for _, job := range t.inMemDb {
		jobs = append(jobs, job)
	}
	return jobs, nil
}

// Initiator ...
type Initiator func(s *jobRepository) *jobRepository

// NewJobRepository ...
func NewJobRepository() Initiator {
	return func(q *jobRepository) *jobRepository {
		q.tokenIndex = make(map[string]*entity.Job)
		return q
	}
}

// SetInMemConnection set database client connection
func (i Initiator) SetInMemConnection(inMemDb map[string]*entity.Job) Initiator {
	return func(s *jobRepository) *jobRepository {
		i(s).inMemDb = inMemDb
		return s
	}
}

// Build ...
func (i Initiator) Build() _interface.JobRepository {
	return i(&jobRepository{})
}
