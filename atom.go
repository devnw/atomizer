package atomizer

import "context"

type Atom interface {
	New() (atom Atom)
	Process(ctx context.Context, ein Electron, eout chan <- Electron) (err error)
	Validate() (valid bool)
}

// TODO: Need to set it up so that an atom can communicate with the original source by sending messages through a channel which takes electrons