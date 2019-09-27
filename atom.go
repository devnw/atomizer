package atomizer

import (
	"context"
)

// Atom is an atomic action with process method for the atomizer to execute the Atom
type Atom interface {
	ID() string
	Process(ctx context.Context, electron Electron, outbound chan<- Electron) (result <-chan []byte)
}

// TODO: Need to set it up so that an atom can communicate with the original source by sending messages through a channel which takes electrons
