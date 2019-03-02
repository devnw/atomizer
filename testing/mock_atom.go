package testing

import "context"

type MockAtom struct {
	id string
	status int
	process func(ctx context.Context, payload []byte) (err error)
}

func (this MockAtom) GetId() (id string) {
	return this.id
}

func (this MockAtom) GetStatus() (status int) {
	return this.status
}

func (this MockAtom) Process(ctx context.Context, payload []byte, estream chan <- MockElectron) (heartbeat <- chan bool, err error) {

	// If this mock atom is mocked with a process function then execute the process function
	// Otherwise just exit
	if this.process != nil {
		err = this.process(ctx, payload)
	}

	return heartbeat, err
}

func (this MockAtom) Panic(ctx context.Context) {
}

func (this MockAtom) Complete(ctx context.Context) (err error) {
	return err
}
