package testing

import "context"

type MockAtom struct {
	process func(ctx context.Context, payload []byte) (err error)
}

func (this MockAtom) Process(ctx context.Context, ein MockElectron, eout chan <- MockElectron) (err error) {

	// If this mock atom is mocked with a process function then execute the process function
	// Otherwise just exit
	if this.process != nil {
		err = this.process(ctx, ein.Payload())
	}

	return err
}

func (this MockAtom) Panic(ctx context.Context) {
}

func (this MockAtom) Complete(ctx context.Context) (err error) {
	return err
}
