package atomizer

import (
	"testing"
)

type testStruct struct{}

// ID returns the unique name of the conductor
func (t *testStruct) ID() string { return "" }

// Receive gets the atoms from the source that are available to atomize
func (t *testStruct) Receive() <-chan []byte { return make(<-chan []byte) }

// Complete mark the completion of an electron instance with applicable statistics
func (t *testStruct) Complete(properties Properties) {}

// Send sends electrons back out through the conductor for additional processing
func (t *testStruct) Send(electron Electron) (result <-chan []byte) { return make(<-chan []byte) }

type invalidTestStruct struct{}

func TestRegister(t *testing.T) {
	tests := []struct {
		key   string
		value interface{}
		err   bool
	}{
		{ // Valid test
			"ValidTest",
			&testStruct{},
			false,
		},
		{ // Invalid test because value is nil
			"NilRegistrationTest",
			nil,
			true,
		},
		{ // Invalid test because value is nil
			"InvalidTypeTest",
			invalidTestStruct{},
			true,
		},
	}

	Clean()

	for _, test := range tests {
		if err := Register(test.key, test.value); err == nil {
			if value, ok := preRegistrations.Load(test.key); ok {
				if _, ok := value.(*testStruct); ok {
					if test.err {
						t.Errorf("Test key [%s] failed because expected a failure but got success", test.key)
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
