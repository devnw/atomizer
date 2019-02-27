package atomizer

type Source interface {

	// Get the atoms from the source that are available to atomize
	GetAtoms() <- chan Atom
}