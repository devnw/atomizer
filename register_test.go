package atomizer

import (
	"testing"
)

type invalidTestStruct struct{}

func TestRegister(t *testing.T) {

	Clean()
	defer Clean()

	tests := []struct {
		key   string
		value interface{}
		err   bool
	}{
		{ // Valid test
			"ValidTest",
			&passthrough{input: make(chan []byte)},
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

	for _, test := range tests {
		if err := Register(nil, test.key, test.value); err == nil {
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
