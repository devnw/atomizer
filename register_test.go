package atomizer

import (
	"github.com/benji-vesterby/validator"
	"sync"
	"testing"
)

type atomTestStruct struct {
	id string
}

func (this atomTestStruct) GetId() (id string) {
	return this.id
}

func (this atomTestStruct) Validate() (valid bool) {
	return len(this.id) > 0
}

func TestRegister(t *testing.T) {
	tests := []struct {
		key   string
		value *atomTestStruct
		err   bool
	}{
		{ // Valid test
			"ValidTest",
			&atomTestStruct{"ValidTest"},
			false,
		},
		{ // Invalid test because key has length of 0
			"",
			&atomTestStruct{"FailKey"},
			true,
		},
		{ // Invalid test because the validate function doesn't allow for empty key
			"Invalid",
			&atomTestStruct{""},
			true,
		},
		{ // Invalid test because the value passed is nil and nil values cannot be registered
			"FailNil",
			nil,
			true,
		},
	}

	atoms = sync.Map{}

	for _, test := range tests {
		if err := register(&atoms, test.key, test.value); err == nil {
			if value, ok := atoms.Load(test.key); ok {
				if atomValue, ok := value.(Atom); ok {
					if validator.IsValid(atomValue) {
						if atomValue.GetId() != test.value.id {
							t.Errorf("Test key [%s] failed because the value stored doesn't match the value returned", test.key)
						}
					} else {
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