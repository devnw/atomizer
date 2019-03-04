package atomizer

import "context"

type Atom interface {
	Process(ctx context.Context, electron Electron, outbound chan <- Electron) (result []byte, err error)
}

// TODO: Need to set it up so that an atom can communicate with the original source by sending messages through a channel which takes electrons