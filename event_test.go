package engine

import (
	"testing"
)

func TestEvent_String(t *testing.T) {
	tests := []struct {
		name     string
		e        *Event
		expected string
	}{
		{
			"message test",
			&Event{
				Message: "test",
			},
			"test",
		},
		{
			"makeEvent message test",
			makeEvent("test"),
			"test",
		},
		{
			"message w/a test",
			&Event{
				Message: "test",
				AtomID:  "10",
			},
			"[aid:10] test",
		},
		{
			"message w/c test",
			&Event{
				Message:     "test",
				ConductorID: "10",
			},
			"[cid:10] test",
		},
		{
			"message w/ea test",
			&Event{
				Message:    "test",
				ElectronID: "10",
				AtomID:     "11",
			},
			"[aid:11 | eid:10] test",
		},
		{
			"message w/eac test",
			&Event{
				Message:     "test",
				ElectronID:  "10",
				AtomID:      "11",
				ConductorID: "12",
			},
			"[cid:12 | aid:11 | eid:10] test",
		},
		{
			"eac test",
			&Event{
				ElectronID:  "10",
				AtomID:      "11",
				ConductorID: "12",
			},
			"[cid:12 | aid:11 | eid:10]",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.e.String()
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
