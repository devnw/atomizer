package atomizer

// Conductor is the interface that should be implemented for passing electrons to the atomizer
// that need processing. This should generally be registered with the atomizer in an initialization script
type Conductor interface {

	// Receive gets the atoms from the source that are available to atomize
	Receive() <-chan Electron

	// Send sends electrons back out through the conductor for additional processing
	Send(electron Electron) (result <-chan []byte)

	// Validate ensure the conductor is valid
	Validate() (valid bool)
}

// Send electron - Ionic
// Share electron? - Covalent
// Split atom
// Quarks - sub atomic sub atomics
