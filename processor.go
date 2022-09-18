// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"context"
	"fmt"
	"io"
	"reflect"

	"go.devnw.com/event"
)

// Processor is an atomic action with process method for the atomizer to execute
// the Processor
type Processor interface {
	Process(context.Context, io.Reader) (io.Reader, error)
}

type pWrapper struct {
	Processor
	*event.Publisher
}

func (p *pWrapper) Process(
	ctx context.Context, data io.Reader,
) (out io.Reader, err error) {
	defer func() {
		r := recover()
		if r != nil {
			err = &Error{
				Msg:   fmt.Sprintf("panic while processing: %v", r),
				Inner: err,
			}
		}
	}()

	out, err = p.Processor.Process(ctx, data)
	return
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
