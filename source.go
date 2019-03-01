package atomizer

type Source interface {

	// Get the atoms from the source that are available to atomize
	GetAtoms() <- chan Atom
}

// Send electron - Ionic
// Share electron? - Covalent
// Split atom
// Quarks - sub atomic sub atomics
