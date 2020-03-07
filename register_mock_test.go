package atomizer

import "sync"

// reset clears out the pre-registered values and channels uses for
// registration this method is primarily for testing and should not be
// used otherwise
func reset() {
	registrant = sync.Map{}
}
