package atomizer

import (
	"context"
)

// mockatom is a struct for mocking Atom in the unit tests
type mockatom struct {
	process func(ctx context.Context, electron Electron, outbound chan<- Electron) (result <-chan []byte, err <-chan error, done <-chan bool)
}

// Process is a method for mocking Atom's process method in the unit tests
func (atom *mockatom) Process(ctx context.Context, electron Electron, outbound chan<- Electron) (result <-chan []byte, err <-chan error, done <-chan bool) {

	// If atom mock atom is mocked with a process function then execute the process function
	// Otherwise just exit
	if atom.process != nil {
		result, err, done = atom.process(ctx, electron, outbound)
	}

	return result, err, done
}
