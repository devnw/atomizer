package atomizer

import (
	"context"
)

// Atom is an atomic action with process method for the atomizer to execute
// the Atom
type Atom interface {
	Process(
		ctx context.Context,
		conductor Conductor,
		electron Electron,
	) ([]byte, error)
}
