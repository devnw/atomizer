package atomizer

// Statuses
const (
	QUEUED = iota
	PUSHED
	PROCESSING
	COMPLETED
	ERRORED
	CANCELLED
)