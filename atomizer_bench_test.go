package atomizer

import (
	"context"
	"testing"

	"github.com/benjivesterby/validator"
)

func BenchmarkAtomizer_Exec_Single(b *testing.B) {
	Clean()

	// Setup a cancellation context for the test so that it has a limited time
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	if conductor, err := harness(ctx); err == nil {

		// cleanup the benchmark timer to get correct measurements
		b.ResetTimer()

		for n := 0; n < b.N; n++ {
			if e, err := newElectron("bench", nil); err == nil {
				// Send the electron onto the conductor
				resp := conductor.Send(ctx, e)

				select {
				case <-ctx.Done():
					b.Error("context closed, test failed")
					return
				case result, ok := <-resp:
					if ok {
						if result != nil && result.Error() == nil {
							// DO NOTHING
						} else {
							b.Error("invalid benchmark")
						}
					} else {
						b.Error("result channel closed, test failed")
					}
				}
			} else {
				b.Errorf("electron creation failure [%s]", err.Error())
			}
		}
	} else {
		b.Errorf("test harness failed [%s]", err.Error())
	}
}

// Benchmarks the creation of an atomizer instance
func BenchmarkAtomize(b *testing.B) {

}

// Benchmarks the cleanup of the atomizer given 1 electron
func BenchmarkAtomizer_Exit1(b *testing.B) {

}

// Benchmarks the cleanup of the atomizer given 10 electrons
func BenchmarkAtomizer_Exit10(b *testing.B) {

}

// Benchmarks the cleanup of the atomizer given 100 electrons
func BenchmarkAtomizer_Exit100(b *testing.B) {

}

// Benchmarks the validation method of the atomizer
func BenchmarkAtomizer_Validate(b *testing.B) {
	var mizer = &atomizer{
		electrons: make(chan instance),
		bonded:    make(chan instance),
		ctx:       context.Background(),
		cancel: context.CancelFunc(func() {

		}),
	}

	for n := 0; n < b.N; n++ {
		if !validator.IsValid(mizer) {
			b.Error("invalid atomizer, expected valid")
		}
	}
}
