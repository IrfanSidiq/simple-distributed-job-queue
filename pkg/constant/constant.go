package constant

type contextKey string

const DataloaderContextKey contextKey = "dataloader"

const (
	JobStatusPending	= "PENDING"
	JobStatusRunning	= "RUNNING"
	JobStatusCompleted	= "COMPLETED"
	JobStatusFailed		= "FAILED"
)

const MaxRetries = 3

const WorkerPoolSize = 10