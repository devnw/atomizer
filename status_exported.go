package atomizer

// TODO: determine if this will ever be used. Othwerwise, re-evaluate if tracking the statues is even necessary
const (
	// PENDING status for bonded atom electron running instance
	PENDING int = iota

	// PROCESSING status for bonded atom electron running instance
	PROCESSING

	// COMPLETED status for bonded atom electron running instance
	COMPLETED
)
