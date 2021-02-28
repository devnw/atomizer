package atomizer

import (
	"context"
	"sync"
	"testing"
	"time"

	"devnw.com/alog"
)

func reset(ctx context.Context, t *testing.T) {
	registrant = sync.Map{}

	if ctx != nil {
		_ = alog.Global(
			ctx,                              // Default context
			"",                               // No prefix
			alog.DEFAULTTIMEFORMAT,           // Standard time format
			time.UTC,                         // UTC logging
			alog.DEFAULTBUFFER,               // Default buffer of 100 logs
			alog.TestDestinations(ctx, t)..., // Default destinations
		)
	}
}

func resetB() {
	registrant = sync.Map{}

	_ = alog.Global(
		context.Background(),        // Default context
		"",                          // No prefix
		alog.DEFAULTTIMEFORMAT,      // Standard time format
		time.UTC,                    // UTC logging
		alog.DEFAULTBUFFER,          // Default buffer of 100 logs
		alog.BenchDestinations()..., // Default destinations
	)
}
