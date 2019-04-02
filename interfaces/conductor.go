package interfaces

// Conductor is the interface that should be implemented for passing electrons to the atomizer
// that need processing. This should generally be registered with the atomizer in an initialization script
type Conductor interface {

	// ID returns the unique name of the conductor
	ID() string

	// Receive gets the atoms from the source that are available to atomize
	Receive() <-chan Electron

	// Complete mark the completion of an electron instance with applicable statistics
	Complete(properties Properties)

	// Send sends electrons back out through the conductor for additional processing
	Send(electron Electron) (result <-chan []byte)

	// Validate ensure the conductor is valid
	Validate() (valid bool)
}

// Send electron - Ionic
// Share electron? - Covalent
// Split atom
// Quarks - sub atomic sub atomics
