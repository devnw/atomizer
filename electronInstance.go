package atomizer

import "context"

type electronInstance struct {
	electron *Electron
	atom *Atom
	ctx context.Context
	cancel context.CancelFunc
}
