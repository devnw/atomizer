// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"context"
)

// Atom is an atomic action with process method for the atomizer to execute
// the Atom
type Atom interface {
	Process(
		ctx context.Context,
		conductor Conductor,
		electron Electron,
	) ([]byte, error)
}
