package atomizer

// Reads in system resource information to determine if there is
// processing capacity available to continue adding new electrons
// to the system or whether or not the initialization of new electrons
// should be temporarily halted
type sampler struct {
	process chan bool
}

func (sampl sampler) sample() {

}

func (sampl sampler) Wait() {

	// Only wait if the process channel has been initialized
	if sampl.process != nil {

		// Block on the channel
		<-sampl.process
	}
}
