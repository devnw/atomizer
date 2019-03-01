package atomizer

import "time"

type Electron interface {
	AtomId() (Id string)
	Payload() (payload []byte)
	Timeout() (timeout *time.Duration)
	Priority() (priority int)
	Validate() (valid bool)

	// TODO: Need to set it up so that an atom can communicate with the original source by sending messages through a channel which takes electrons
	Callback() (callback func(results []byte) (err error))
}