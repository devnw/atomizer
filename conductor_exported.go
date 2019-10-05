package atomizer

import "context"

// Conductor is the interface that should be implemented for passing electrons to the atomizer
// that need processing. This should generally be registered with the atomizer in an initialization script
type Conductor interface {

	// ID returns the unique name of the conductor
	ID() string

	// Receive gets the atoms from the source that are available to atomize
	Receive(ctx context.Context) <-chan []byte

	// Complete mark the completion of an electron instance with applicable statistics
	Complete(ctx context.Context, properties Properties) error

	// Send sends electrons back out through the conductor for additional processing
	Send(ctx context.Context, electron Electron) (result <-chan Properties)
}

// Send electron - Ionic
// Share electron? - Covalent
// Split atom
// Quarks - sub atomic sub atomics
