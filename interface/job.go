package _interface

import (
	"context"
	"jobqueue/entity"
)

type JobService interface {
	Enqueue(ctx context.Context, taskName string) (*entity.Job, error)
	GetJobByID(ctx context.Context, id string) (*entity.Job, error)
	GetAllJobs(ctx context.Context) ([]*entity.Job, error)
	GetAllJobsStatus(ctx context.Context) (*entity.JobStatus, error)
}

type JobRepository interface {
	Save(ctx context.Context, job *entity.Job) error
	FindByID(ctx context.Context, id string) (*entity.Job, error)
	FindAll(ctx context.Context) ([]*entity.Job, error)
}
