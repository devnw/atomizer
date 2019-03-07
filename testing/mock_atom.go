package testing

import (
	"context"
	"github.com/benji-vesterby/atomizer"
)

// MockAtom is a struct for mocking Atom in the unit tests
type MockAtom struct {
	process func(ctx context.Context, payload []byte) (err error)
}

// Process is a method for mocking Atom's process method in the unit tests
func (atom *MockAtom) Process(ctx context.Context, electron atomizer.Electron, outbound chan<- atomizer.Electron) (result []byte, err error) {

	// If atom mock atom is mocked with a process function then execute the process function
	// Otherwise just exit
	if atom.process != nil {
		err = atom.process(ctx, electron.Payload())
	}

	return result, err
}
