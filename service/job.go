package service

import (
	"context"
	"fmt"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"jobqueue/pkg/constant"
	"log"
	"time"

	"github.com/google/uuid"
)

type jobService struct {
	jobRepo _interface.JobRepository
}

type Initiator func(s *jobService) *jobService

func (s *jobService) Enqueue(ctx context.Context, taskName string) (*entity.Job, error) {
	jobID := uuid.NewString()
	job := &entity.Job{
		ID:			jobID,
		Task:		taskName,
		Status:		constant.JobStatusPending,
		Attempts: 	0,
	}

	err := s.jobRepo.Save(ctx, job)
	if err != nil {
		log.Printf("Error saving job %s: %v\n", jobID, err)
		return nil, fmt.Errorf("failed to save job: %w", err)
	}
	log.Printf("Job enqueued: ID=%s, Task=%s\n", job.ID, job.Task)

	go s.processJob(job)

	return job, nil
}

func (s *jobService) processJob(initialJob *entity.Job) {
	job, err := s.jobRepo.FindByID(context.Background(), initialJob.ID)
	if err != nil {
		log.Printf("Error fetching job %s for processing: %v\n", initialJob.ID, err)
		return
	}

	log.Printf("Processing job: ID=%s, Task=%s, Attempt=%d\n", job.ID, job.Task, job.Attempts+1)

	job.Status = constant.JobStatusRunning
	job.Attempts++
	if err := s.jobRepo.Save(context.Background(), job); err != nil {
		log.Printf("Error updating job %s to RUNNING: %v\n", job.ID, err)
		return
	}

	processingTime := 3 * time.Second
	isUnstableJob := job.Task == "unstable-job"

	if isUnstableJob {
		processingTime = 1 * time.Second
		if job.Attempts <= 2 {
			time.Sleep(processingTime)
			job.Status = constant.JobStatusFailed
			log.Printf("Job FAILED (unstable): ID=%s, Task=%s, Attempt=%d\n", job.ID, job.Task, job.Attempts)

			if err := s.jobRepo.Save(context.Background(), job); err != nil {
				log.Printf("Error updating job %s to FAILED (unstable): %v\n", job.ID, err)
			}

			if job.Attempts < constant.MaxRetries {
				log.Printf("Retrying unstable job: ID=%s, Attempt %d of %d\n", job.ID, job.Attempts, constant.MaxRetries)
				time.Sleep(2 * time.Second)
				s.processJob(job)
			} else {
				log.Printf("Unstable job %s reached max retries and still failed.\n", job.ID)
			}
			return
		}
	}

	time.Sleep(processingTime)

	job.Status = constant.JobStatusCompleted
	log.Printf("Job COMPLETED: ID=%s, Task=%s, Attempt=%d\n", job.ID, job.Task, job.Attempts)
	if err := s.jobRepo.Save(context.Background(), job); err != nil {
		log.Printf("Error updating job %s to COMPLETED: %v\n", job.ID, err)
	}
}

func (s *jobService) GetJobByID(ctx context.Context, id string) (*entity.Job, error) {
	job, err := s.jobRepo.FindByID(ctx, id)
	if err != nil {
		log.Printf("Error finding job by ID %s: %v\n", id, err)
		return nil, fmt.Errorf("job with ID %s not found: %w", id, err)
	}
	return job, nil
}

func (s *jobService) GetAllJobs(ctx context.Context) ([]*entity.Job, error) {
	jobs, err := s.jobRepo.FindAll(ctx)
	if err != nil {
		log.Printf("Error getting all jobs: %v\n", err)
		return nil, fmt.Errorf("failed to get all jobs: %w", err)
	}
	return jobs, nil
}

func (s *jobService) GetAllJobsStatus(ctx context.Context) (*entity.JobStatus, error) {
	jobs, err := s.jobRepo.FindAll(ctx)
	if err != nil {
		log.Printf("Error getting all jobs for status calculation: %v\n", err)
		return nil, fmt.Errorf("failed to get jobs for status: %w", err)
	}

	statusCounts := &entity.JobStatus{}
	for _, job := range jobs {
		switch job.Status {
		case constant.JobStatusPending:
			statusCounts.Pending++
		case constant.JobStatusRunning:
			statusCounts.Running++
		case constant.JobStatusFailed:
			statusCounts.Failed++
		case constant.JobStatusCompleted:
			statusCounts.Completed++
		}
	}

	return statusCounts, nil
}

func NewJobService() Initiator {
	return func(s *jobService) *jobService {
		return s
	}
}

func (i Initiator) SetJobRepository(jobRepository _interface.JobRepository) Initiator {
	return func(s *jobService) *jobService {
		i(s).jobRepo = jobRepository
		return s
	}
}

func (i Initiator) Build() _interface.JobService {
	return i(&jobService{})
}