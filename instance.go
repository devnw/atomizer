package atomizer

import "context"

type instance struct {
	electron Electron
	atom Atom
	outbound <- chan Electron
	ctx context.Context
	cancel context.CancelFunc
}
