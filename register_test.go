package engine

import (
	"testing"
)

type invalidTestStruct struct{}

func TestRegister(t *testing.T) {
	ctx, cancel := _ctx(nil)
	defer cancel()

	tests := []struct {
		name  string
		key   string
		value interface{}
		err   bool
	}{
		{
			"valid conductor registration",
			ID(noopconductor{}),
			&noopconductor{},
			false,
		},
		{
			"valid atom registration",
			ID(noopatom{}),
			&noopatom{},
			false,
		},
		{
			"invalid nil registration",
			"",
			nil,
			true,
		},
		{
			"invalid interface registration",
			ID(invalidTestStruct{}),
			invalidTestStruct{},
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reset(ctx, t)
			defer reset(nil, t)

			err := Register(test.value)
			if err != nil && !test.err {
				t.Fatal(err)
			}

			_, ok := registrant.Load(test.key)
			if !ok && !test.err {
				t.Fatalf(
					"Test key [%s] failed to load",
					test.key,
				)
			}
		})
	}
}
