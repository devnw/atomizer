package engine

import (
	"testing"

	"devnw.com/validator"
	"github.com/pkg/errors"
)

func TestError_Error(t *testing.T) {
	prefix := "atomizer error"

	tests := []struct {
		name     string
		e        error
		expected string
	}{
		{
			"error w/message test",
			Error{
				Event: Event{
					Message: "test",
				},
			},
			prefix + " test",
		},
		{
			"error w/empty message test",
			Error{
				Event: Event{},
			},
			prefix,
		},
		{
			"error w/inner error test",
			Error{
				Internal: Error{
					Event: Event{
						Message: "test",
					},
				},
			},
			prefix + " | internal: (atomizer error test)",
		},
		{
			"error w/multiple inner error test",
			Error{
				Internal: Error{
					Event: Event{
						Message: "test",
					},
					Internal: Error{
						Event: Event{
							Message: "test 2",
						},
					},
				},
			},
			prefix + " | internal: (atomizer error test" +
				" | internal: (atomizer error test 2))",
		},
		{
			"simple error test",
			simple("test", nil),
			prefix + " test",
		},
		{
			"simple error w/inner test",
			simple(
				"test",
				simple("inner", nil),
			),
			prefix + " test | internal: (atomizer error inner)",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.e.Error()
			if result != test.expected {
				t.Fatalf(
					"expected [%s] got [%s]",
					test.expected,
					result,
				)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {

	e := errors.New("wrapped error")

	aerr := Error{
		Internal: Error{
			Internal: e,
		},
	}

	if aerr.Unwrap() != e {
		t.Fatalf(
			"expected [%v] got [%v]",
			e,
			aerr.Unwrap(),
		)
	}
}

func TestError_Unwrap_Fail(t *testing.T) {

	e := errors.New("wrapped error")

	aerr := Error{
		Internal: e,
	}

	if aerr.Unwrap() != e {
		t.Fatalf(
			"expected [%v] got [%v]",
			e,
			aerr.Unwrap(),
		)
	}
}

func TestError_Validate(t *testing.T) {

	tests := []struct {
		name  string
		e     error
		valid bool
	}{
		{
			"error w/valid event",
			Error{
				Event: Event{
					Message: "test",
				},
			},
			true,
		},
		{
			"error w/invalid event",
			Error{
				Event: Event{},
			},
			false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v := validator.Valid(test.e)
			if test.valid != v {
				t.Fatalf(
					"expected [%v] got [%v]",
					test.valid,
					v,
				)
			}
		})
	}
}
