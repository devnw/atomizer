package atomizer

// Conductor is the interface that should be implemented for passing electrons to the atomizer
// that need processing. This should generally be registered with the Atomizer in an initialization script
type Conductor interface {

	// Get the atoms from the source that are available to atomize
	Receive() <-chan Electron
	Send(electron Electron) (result <-chan []byte)
	Validate() (valid bool)
}

// Send electron - Ionic
// Share electron? - Covalent
// Split atom
// Quarks - sub atomic sub atomics
