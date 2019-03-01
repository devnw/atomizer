package atomizer

type Conductor interface {

	// Get the atoms from the source that are available to atomize
	Receive() <- chan Electron
	Send(electron Electron)
	Validate() (valid bool)
}

// Send electron - Ionic
// Share electron? - Covalent
// Split atom
// Quarks - sub atomic sub atomics
