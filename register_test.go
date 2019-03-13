package atomizer

import (
	"context"
	"sync"
	"testing"

	"github.com/benji-vesterby/validator"
)

type atomTestStruct struct {
}

func (atomteststr *atomTestStruct) Validate() (valid bool) {
	return atomteststr != nil
}

func (atomteststr *atomTestStruct) Process(ctx context.Context, electron Electron, outbound chan<- Electron) (result <-chan []byte, err <-chan error, done <-chan bool) {
	return result, err, done
}

type nonatomtestregister struct {
	id string
}

func (nonatomtestreg *nonatomtestregister) Validate() (valid bool) {
	return len(nonatomtestreg.id) > 0
}

func TestRegister(t *testing.T) {
	tests := []struct {
		key   string
		value interface{}
		err   bool
	}{
		{ // Valid test
			"ValidTest",
			&atomTestStruct{},
			false,
		},
		{ // Invalid test because key has length of 0
			"",
			&atomTestStruct{},
			true,
		},
		{ // Invalid test because the value passed is nil and nil values cannot be registered
			"FailNil",
			nil,
			true,
		},
		{ // Invalid test because the struct doesn't implement atom
			"wronginterface",
			&nonatomtestregister{},
			true,
		},
	}

	atoms = sync.Map{}

	for _, test := range tests {
		if err := register(&atoms, test.key, test.value); err == nil {
			if value, ok := atoms.Load(test.key); ok {
				if atomValue, ok := value.(Atom); ok {
					if !validator.IsValid(atomValue) {
						t.Errorf("Test key [%s] failed because the returned value was invalid", test.key)
					}
				} else {
					t.Errorf("Test key [%s] failed because returned value failed type assertion", test.key)
				}
			} else {
				t.Errorf("Test key [%s] failed to load value from sync map", test.key)
			}
		} else if err != nil && !test.err {
			t.Error(err)
		}
	}
}

func TestRegisterSource(t *testing.T) {

}

func TestRegisterAtom(t *testing.T) {

}
