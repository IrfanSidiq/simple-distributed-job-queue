package mutation

import (
	"context"
	_dataloader "jobqueue/delivery/graphql/dataloader"
	"jobqueue/delivery/graphql/resolver"
	_interface "jobqueue/interface"
	"log"
)

type JobMutation struct {
	jobService _interface.JobService
	dataloader *_dataloader.GeneralDataloader
}

func (q JobMutation) Enqueue(ctx context.Context, args struct{ Task string }) (*resolver.JobResolver, error) {
	log.Printf("GraphQL Enqueue called with task: %s\n", args.Task)
	createdJob, err := q.jobService.Enqueue(ctx, args.Task)
	if err != nil {
		log.Printf("Error in JobMutation.Enqueue: %v\n", err)
		return nil, err
	}

	return &resolver.JobResolver{
		Data:       *createdJob,
		JobService: q.jobService,
		Dataloader: q.dataloader,
	}, nil
}

// NewJobMutation to create new instance
func NewJobMutation(jobService _interface.JobService, dataloader *_dataloader.GeneralDataloader) JobMutation {
	return JobMutation{
		jobService: jobService,
		dataloader: dataloader,
	}
}
