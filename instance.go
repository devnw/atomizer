package atomizer

import "context"

type instance struct {
	ewrap ewrappers
	atom Atom
	outbound <- chan Electron
	ctx context.Context
	cancel context.CancelFunc
}
