package atomizer

import (
	"context"

	"github.com/pkg/errors"
)

// type ewrappers struct {
// 	electron  Electron
// 	conductor Conductor
// 	// ctx       context.Context
// 	// cancel    context.CancelFunc
// }

// func (w *ewrappers) Cancel() (err error) {
// 	return cancel(w.cancel)
// }

type cwrapper struct {
	Conductor
	ctx    context.Context
	cancel context.CancelFunc
}

func (w *cwrapper) Cancel() (err error) {
	return wrapcancel(w.cancel)
}

type awrapper struct {
	Atom
	ctx    context.Context
	cancel context.CancelFunc
}

func (w *awrapper) Cancel() (err error) {
	return wrapcancel(w.cancel)
}

type cancelable interface {
	Cancel() error
}

func wrapcancel(c context.CancelFunc) (err error) {

	if c != nil {
		c()
	} else {
		err = errors.New("cancel function is nil")
	}

	return err
}
