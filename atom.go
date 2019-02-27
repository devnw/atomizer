package atomizer

import "context"

type Atom interface {
	GetId() (id string)
	GetStatus() (status int)
	Process(ctx context.Context, payload []byte) (err error)
	Panic(ctx context.Context)
	Complete(ctx context.Context) (err error)
}