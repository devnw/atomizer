package atomizer

import (
	"context"
)

// mockatom is a struct for mocking Atom in the unit tests
type mockatom struct {
	process func(ctx context.Context, payload []byte) (err error)
}

// Process is a method for mocking Atom's process method in the unit tests
func (atom *mockatom) Process(ctx context.Context, electron Electron, outbound chan<- Electron) (result []byte, err error) {

	// If atom mock atom is mocked with a process function then execute the process function
	// Otherwise just exit
	if atom.process != nil {
		err = atom.process(ctx, electron.Payload())
	}

	return result, err
}
