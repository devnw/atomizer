package testing

import "time"

type MockElectron struct {

}

func (this MockElectron) Atom() (Id string) {

}

func (this MockElectron) Id() (Id string) {

}

func (this MockElectron) Payload() (payload []byte) {

}

func (this MockElectron) Timeout() (timeout *time.Duration) {

}

func (this MockElectron) Priority() (priority int) {

}

func (this MockElectron) Validate() (valid bool) {

}
