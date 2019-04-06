package atomizer

import (
	"context"
	"reflect"

	"github.com/pkg/errors"

	"github.com/benji-vesterby/atomizer/interfaces"
)

type ewrappers struct {
	electron  interfaces.Electron
	conductor interfaces.Conductor
	// ctx       context.Context
	// cancel    context.CancelFunc
}

// func (w *ewrappers) Cancel() (err error) {
// 	return cancel(w.cancel)
// }

type cwrapper struct {
	interfaces.Conductor
	ctx    context.Context
	cancel context.CancelFunc
}

func (w *cwrapper) Cancel() (err error) {
	return cancel(w.cancel)
}

type awrapper struct {
	interfaces.Atom
	ctx    context.Context
	cancel context.CancelFunc
}

func (w *awrapper) Cancel() (err error) {
	return cancel(w.cancel)
}

type cancelable interface {
	Cancel() error
}

func cancel(c context.CancelFunc) (err error) {

	if c != nil {
		c()
	} else {
		err = errors.New("cancel function is nil")
	}

	return err
}

func nuke(obj interface{}) (err error) {

	if c, ok := obj.(cancelable); ok {
		// Cancel the object
		err = c.Cancel()

	} else {
		err = errors.Errorf("unable to nuke [%v], invalid cancelable obj", reflect.TypeOf(obj))
	}

	return err
}
