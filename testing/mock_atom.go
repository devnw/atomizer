package testing

import (
	"context"
	"github.com/benji-vesterby/atomizer"
)

type MockAtom struct {
	process func(ctx context.Context, payload []byte) (err error)
}

func (this *MockAtom) Process(ctx context.Context, electron atomizer.Electron, outbound chan <- atomizer.Electron)  (result []byte, err error) {

	// If this mock atom is mocked with a process function then execute the process function
	// Otherwise just exit
	if this.process != nil {
		err = this.process(ctx, electron.Payload())
	}

	return result, err
}