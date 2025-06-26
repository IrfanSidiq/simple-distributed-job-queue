package query

import (
	"context"
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/delivery/graphql/resolver"
	"jobqueue/entity"
	_interface "jobqueue/interface"
	"log"
)

type JobQuery struct {
	jobService _interface.JobService
	dataloader *_dataloader.GeneralDataloader
}

func (q JobQuery) Jobs(ctx context.Context) ([]resolver.JobResolver, error) {
	log.Println("GraphQL Jobs query called")
	jobs, err := q.jobService.GetAllJobs(ctx)
	if err != nil {
		log.Printf("Error in JobQuery.Jobs: %v\n", err)
		return nil, err
	}

	resolvers := make([]resolver.JobResolver, len(jobs))
	for i, job := range jobs {
		resolvers[i] = resolver.JobResolver{
			Data:		*job,
			JobService: q.jobService,
			Dataloader: q.dataloader,
		}
	}

	return resolvers, nil
}

func (q JobQuery) Job(ctx context.Context, args struct { ID string }) (*resolver.JobResolver, error) {
	log.Printf("GraphQL Job query called with ID: %s\n", args.ID)
	job, err := q.jobService.GetJobByID(ctx, args.ID)
	if err != nil {
		log.Printf("Error in JobQuery.Job: %v\n", err)
		return nil, err
	}
	if job == nil {
		return nil, nil
	}

	return &resolver.JobResolver{
		Data:		*job,
		JobService: q.jobService,
		Dataloader: q.dataloader,
	}, nil
}

func (q JobQuery) JobStatus(ctx context.Context) (*resolver.JobStatusResolver, error) {
	log.Println("GraphQL JobStatus query called")
	status, err := q.jobService.GetAllJobsStatus(ctx)
	if err != nil {
		log.Printf("Error in JobQuery.JobStatus: %v\n", err)
		return nil, err
	}
	if status == nil {
		return &resolver.JobStatusResolver{
			Data:       entity.JobStatus{},
			JobService: q.jobService,
			Dataloader: q.dataloader,
		}, nil
	}

	return &resolver.JobStatusResolver{
		Data:       *status,
		JobService: q.jobService,
		Dataloader: q.dataloader,
	}, nil
}


func NewJobQuery(jobService _interface.JobService, dataloader *_dataloader.GeneralDataloader) JobQuery {
	return JobQuery{
		jobService: jobService,
		dataloader: dataloader,
	}
}
