// Copyright © 2019 Developer Network, LLC
//
// This file is subject to the terms and conditions defined in
// file 'LICENSE', which is part of this source code package.

package engine

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// _ctx returns a valid context with cancel even if it the
// supplied context is initially nil. If the supplied context
// is nil it uses the background context
func _ctx(c context.Context) (context.Context, context.CancelFunc) {
	if c == nil {
		c = context.Background()
	}

	return context.WithCancel(c)
}

// _ctxT returns a context with a timeout that is passed in as a time.Duration
func _ctxT(
	c context.Context,
	duration *time.Duration,
) (context.Context, context.CancelFunc) {
	if c == nil {
		c = context.Background()
	}

	if duration == nil {
		return _ctx(c)
	}

	return context.WithTimeout(c, *duration)
}

// ID returns the registration id for the passed in object type
func ID(v any) string {
	return strings.Trim(fmt.Sprintf("%T", v), "*")
}
