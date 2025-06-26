package service

import (
	"context"
	"fmt"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"jobqueue/pkg/constant"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type jobService struct {
	jobRepo _interface.JobRepository
	jobsChannel chan *entity.Job
	wg          sync.WaitGroup
}

type Initiator func(s *jobService) *jobService

func NewJobService() Initiator {
	return func(s *jobService) *jobService {
		s.jobsChannel = make(chan *entity.Job, constant.WorkerPoolSize * 2)
		s.startWorkers(constant.WorkerPoolSize)
		log.Printf("%d job workers started.\n", constant.WorkerPoolSize)
		return s
	}
}

func (s *jobService) startWorkers(numWorkers int) {
	for i := 0; i < numWorkers; i++ {
		workerID := i + 1
		s.wg.Add(1)
		go func(id int) {
			defer s.wg.Done()
			log.Printf("Worker %d started and listening for jobs.\n", id)
			for job := range s.jobsChannel {
				log.Printf("Worker %d picked up job: ID=%s, Task=%s\n", id, job.ID, job.Task)
				s.processJob(job)
			}
			log.Printf("Worker %d shutting down (jobsChannel closed).\n", id)
		}(workerID)
	}
}

func (s *jobService) Shutdown() {
	log.Println("Job Service shutting down. No more jobs will be enqueued.")
	close(s.jobsChannel)

	log.Println("Waiting for all workers to finish their current jobs...")
	s.wg.Wait()
	log.Println("All workers have finished. Job service shutdown complete.")
}

func (s *jobService) Enqueue(ctx context.Context, taskName string, token *string) (*entity.Job, error) {
	if token != nil && *token != "" {
		existingJob, err := s.jobRepo.FindByToken(ctx, *token)
		if err == nil && existingJob != nil {
			log.Printf("Token '%s' matched existing job ID '%s'. Returning existing job.", *token, existingJob.ID)
			return existingJob, nil
		}
	}

	jobID := uuid.NewString()
	job := &entity.Job{
		ID:             jobID,
		Token: 			token,
		Task:           taskName,
		Status:         constant.JobStatusPending,
		Attempts:       0,
	}

	err := s.jobRepo.Save(ctx, job)
	if err != nil {
		log.Printf("Error saving job %s: %v\n", jobID, err)
		return nil, fmt.Errorf("failed to save job: %w", err)
	}
	log.Printf("New job enqueued: ID=%s, Task=%s.\n", job.ID, job.Task)

	s.jobsChannel <- job

	return job, nil
}

func (s *jobService) processJob(jobToProcess *entity.Job) {
	job, err := s.jobRepo.FindByID(context.Background(), jobToProcess.ID)
	if err != nil {
		log.Printf("Error fetching job %s for processing by worker: %v\n", jobToProcess.ID, err)
		return
	}

	if job.Status == constant.JobStatusCompleted || (job.Status == constant.JobStatusFailed && job.Attempts >= constant.MaxRetries) {
		log.Printf("Job %s already processed (Status: %s, Attempts: %d). Skipping.\n", job.ID, job.Status, job.Attempts)
		return
	}

	log.Printf("Processing job: ID=%s, Task=%s, CurrentAttempt=%d (max %d)\n", job.ID, job.Task, job.Attempts, constant.MaxRetries)

	job.Attempts++
	job.Status = constant.JobStatusRunning
	log.Printf("Processing job: ID=%s, Task=%s, Attempt=%d\n", job.ID, job.Task, job.Attempts)

	if err := s.jobRepo.Save(context.Background(), job); err != nil {
		log.Printf("Error updating job %s to RUNNING (Attempt %d): %v\n", job.ID, job.Attempts, err)
		return
	}


	processingTime := 3 * time.Second
	isUnstableJob := job.Task == "unstable-job"
	taskSucceeded := true

	if isUnstableJob {
		processingTime = 1 * time.Second
		if job.Attempts <= 2 {
			taskSucceeded = false
		}
	}

	time.Sleep(processingTime)

	if taskSucceeded {
		job.Status = constant.JobStatusCompleted
		log.Printf("Job COMPLETED: ID=%s, Task=%s, Attempt=%d\n", job.ID, job.Task, job.Attempts)
	} else {
		if job.Attempts < constant.MaxRetries {
			log.Printf("Job FAILED (will retry): ID=%s, Task=%s, Attempt=%d of %d\n", job.ID, job.Task, job.Attempts, constant.MaxRetries)
			job.Status = constant.JobStatusPending
			log.Printf("Unstable job %s failed on attempt %d. Marked PENDING for retry.\n", job.ID, job.Attempts)

			if err := s.jobRepo.Save(context.Background(), job); err != nil {
				log.Printf("Error saving unstable job %s as PENDING for retry: %v\n", job.ID, err)
				return
			}

			time.Sleep(2 * time.Second)
			log.Printf("Re-queueing unstable job %s for attempt %d\n", job.ID, job.Attempts+1)

			s.jobsChannel <- job
			return
		} else {
			job.Status = constant.JobStatusFailed
			log.Printf("Job FAILED (max retries reached): ID=%s, Task=%s, Attempt=%d\n", job.ID, job.Task, job.Attempts)
		}
	}

	if err := s.jobRepo.Save(context.Background(), job); err != nil {
		log.Printf("Error updating job %s to final status %s: %v\n", job.ID, job.Status, err)
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

func (i Initiator) SetJobRepository(jobRepository _interface.JobRepository) Initiator {
	return func(s *jobService) *jobService {
		i(s).jobRepo = jobRepository
		return s
	}
}

func (i Initiator) Build() _interface.JobService {
	return i(&jobService{})
}