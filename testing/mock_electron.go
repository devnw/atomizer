package testing

import "time"

type MockElectron struct {

}

func (this MockElectron) Atom() (Id string) {
	return Id
}

func (this MockElectron) Id() (Id string) {
	return Id
}

func (this MockElectron) Payload() (payload []byte) {
	return payload
}

func (this MockElectron) Timeout() (timeout *time.Duration) {
	return timeout
}

func (this MockElectron) Validate() (valid bool) {

	return valid
}

func (this MockElectron) Callback(result []byte) (err error) {

	return err
}