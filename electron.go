package atomizer

import "time"

type Electron interface {
	AtomId() (Id string)
	Payload() (payload []byte)
	Timeout() (timeout *time.Duration)
	Priority() (priority int)
}