package atomizer

import (
	"testing"
)

type invalidTestStruct struct{}

func TestRegister(t *testing.T) {

	Clean()
	defer Clean()

	tests := []struct {
		title string
		key   string
		value interface{}
		err   bool
	}{
		{ // Valid test
			"ValidTest",
			"atomizer.passthrough",
			&passthrough{input: make(chan *Electron)},
			false,
		},
		{ // Invalid test because value is nil
			"NilRegistrationTest",
			"nil",
			nil,
			true,
		},
		{ // Invalid test because value is nil
			"InvalidTypeTest",
			"atomizer.invalidTestStruct",
			invalidTestStruct{},
			true,
		},
	}

	for _, test := range tests {
		if err := Register(nil, test.value); err == nil {
			if value, ok := preRegistrations.Load(test.key); ok {
				if _, ok := value.(*passthrough); ok {
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
