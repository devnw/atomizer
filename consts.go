package atomizer

// Priorities
const (
	CRITICAL = iota
	HIGH
	MEDIUM
	LOW
)

// Statuses
const (
	QUEUED = iota
	PUSHED
	PROCESSING
	COMPLETED
	ERRORED
	CANCELLED
)