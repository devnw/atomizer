package atomizer

import "context"

type Atom interface {
	GetId() (id string)
	GetStatus() (status int)
	Process(ctx context.Context, payload []byte) (heartbeat <- chan bool, err error)
	Validate() (valid bool)
}