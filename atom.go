package atomizer

import "context"

type Atom interface {
	GetId() (id string)
	GetStatus() (status int)
	Process(ctx context.Context, payload []byte, estream chan <- Electron) (heartbeat <- chan bool, err error)
	Validate() (valid bool)
}

// TODO: Need to set it up so that an atom can communicate with the original source by sending messages through a channel which takes electrons