package atomizer

import (
	"context"

	"github.com/benji-vesterby/atomizer/interfaces"
)

type instance struct {
	ewrap    ewrappers
	atom     interfaces.Atom
	outbound <-chan interfaces.Electron
	ctx      context.Context
	cancel   context.CancelFunc
	// TODO: add an actions channel here that the monitor can keep an eye on for this bonded electron/atom combo
}

func (inst *instance) Validate() (valid bool) {
	return valid
}
