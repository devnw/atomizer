package atomizer

import (
	"context"
	"sync"
	"time"

	"github.com/shirou/gopsutil/mem"
)

// Reads in system resource information to determine if there is
// processing capacity available to continue adding new electrons
// to the system or whether or not the initialization of new electrons
// should be temporarily halted
type sampler struct {
	process chan bool
	once    *sync.Once
	ctx     context.Context
	cancel  context.CancelFunc
}

func (sampl sampler) sample() {

	go func() {

		var subctx context.Context

		for {
			select {
			case <-sampl.ctx.Done():
				return
			default:
				if v, err := mem.VirtualMemory(); err == nil {

					// TODO: Work on this
					if v.UsedPercent <= 100 {
						// reset the sub context so the timeout goes away
						subctx = nil

						//establishes to wait if the used memory percentage is at a set amount
						select {
						case <-sampl.ctx.Done():
							return
						case sampl.process <- true:
						}
					} else {
						if subctx == nil {
							var cancel context.CancelFunc
							subctx, cancel = context.WithTimeout(sampl.ctx, time.Second*300)
							defer cancel()
						}

						select {
						case <-subctx.Done():
							panic("sampler timed out without processing for x minutes")
						}
					}
				}
				<-time.After(5 * time.Millisecond)
			}
		}
	}()

}

func (sampl sampler) Wait() {
	sampl.once.Do(sampl.sample)
	// Block on the channel
	<-sampl.process
}
