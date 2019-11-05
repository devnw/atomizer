package atomizer

import (
	"context"
	"github.com/shirou/gopsutil/mem"
	"sync"
	"time"
)

// Reads in system resource information to determine if there is
// processing capacity available to continue adding new electrons
// to the system or whether or not the initialization of new electrons
// should be temporarily halted
type sampler struct {
	process chan bool
	once    *sync.Once
}

func (sampl sampler) sample() {
	v, _ := mem.VirtualMemory()
	usedMem := v.UsedPercent

	if usedMem >= 70.00 {
		//establishes to wait if the used memory percentage is at a set amount
	} else {
		//what am I returning here to the instance?
	}

}

func (sampl sampler) Wait(ctx context.Context) {
	sampl.once.Do(sampl.sample)
	// Only wait if the process channel has been initialized
	if sampl.process != nil {
		select {
		case <-ctx.Done():
			return
		case <-time.After(5 * time.Second):
			sampl.sample()
		}
		// Block on the channel
		<-sampl.process
	}
}
