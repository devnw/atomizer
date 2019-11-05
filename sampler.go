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
}

func (sampl sampler) sample() {
	go func() {
		for {
			select {
			case <-sampl.ctx.Done():
				return
			default:
				if v, err := mem.VirtualMemory(); err == nil {
					if v.UsedPercent <= 70.00 {
						//establishes to wait if the used memory percentage is at a set amount
						select {
						case <-sampl.ctx.Done():
							return
						case sampl.process <- true:
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
