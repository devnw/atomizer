// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import "context"

// Conductor is the interface that should be implemented for passing
// electrons to the atomizer that need processing. This should generally be
// registered with the atomizer in an initialization script
type Conductor interface {

	// Receive gets the atoms from the source
	// that are available to atomize
	Receive(ctx context.Context) <-chan *Electron

	// Complete mark the completion of an electron instance
	// with applicable statistics
	Complete(ctx context.Context, p *Properties) error

	// Send sends electrons back out through the conductor for
	// additional processing
	Send(ctx context.Context, electron *Electron) (<-chan *Properties, error)

	// Close cleans up the conductor
	Close()
}
