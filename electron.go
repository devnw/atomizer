package atomizer

import "time"

type ewrappers struct {
	electron  Electron
	conductor Conductor
}

// Electron is the interface which should be implemented by the messages sent through the conductors that trigger
// instances of each atom for processing.
type Electron interface {
	Atom() (ID string)
	ID() (ID string)
	Payload() (payload []byte)
	Timeout() (timeout *time.Duration)
	Validate() (valid bool)

	// Callback allows the system to return results of a spawned electron to the caller after it's been bonded
	// TODO: Need to set it up so that an atom can communicate with the original source by sending messages through a channel which takes electrons
	//  When the electron is sent back to another node a channel is opened by the send method of the source and blocking will occur on reading from that channel
	//  rather than relying on a callback with a waitgroup which is less reliable
	Callback(result []byte) (err error)
}
