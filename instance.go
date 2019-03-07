package atomizer

import "context"

type instance struct {
	ewrap    ewrappers
	atom     Atom
	outbound <-chan Electron
	ctx      context.Context
	cancel   context.CancelFunc
	// TODO: add an actions channel here that the monitor can keep an eye on for this bonded electron/atom combo
}

func (inst *instance) Validate() (valid bool) {
	return valid
}
