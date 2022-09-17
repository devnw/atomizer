// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"context"
	"reflect"

	"go.devnw.com/event"
)

// Processor is an atomic action with process method for the atomizer to execute
// the Processor
type Processor interface {
	Process(context.Context, []byte) ([]byte, error)
}

type pWrapper struct {
	Processor
	*event.Publisher
}

func (p *pWrapper) Process(
	ctx context.Context, data []byte,
) ([]byte, error) {
	out, err := p.Processor.Process(ctx, data)
	if err != nil {
		return out, err
	}

	return out, err
}

type maker struct {
	proc Processor
}

func (m *maker) init() Processor {
	// Initialize a new copy of the atom
	newAtom := reflect.New(
		reflect.TypeOf(m.proc).Elem(),
	)

	// ok is not checked here because this should
	// never fail since the originating data item
	// is what created this
	out, _ := newAtom.Interface().(Processor)

	return out
}

func (m *maker) Make(ctx context.Context) <-chan Processor {
	out := make(chan Processor)

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				return
			case out <- m.init():

			}
		}
	}()

	return out
}
