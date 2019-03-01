package atomizer

import "context"

type Atomizer struct {

	// Priority Channels
	Critical chan Electron
	High chan Electron
	Medium chan Electron
	Low chan Electron
}

func (this Atomizer) Receive(ctx context.Context)  {
	for {
		select {
			case <- ctx.Done():
				// TODO:
				break
		default:
			conductors.Range(func(key, value interface{}) bool {
				var ok bool

				var conductor Conductor
				if conductor, ok = value.(Conductor); ok {
					if err := this.FanOut(ctx, conductor.Receive()); err == nil {
						ok = true
					} else {
						// TODO: Decide how to handle the error here
					}
				}

				return ok
			})
		}
	}
}

func (this Atomizer) FanOut(ctx context.Context, electrons <- chan Electron) (err error) {

	if ctx != nil {
		if electrons != nil {

			go func() {
				for {
					select {
					case <- ctx.Done():
					// Do nothing, and let the default false ok break the loop
					case electron := <- electrons:
						switch electron.Priority() {
						case CRITICAL:
							this.Critical <- electron
						case HIGH:
							this.High <- electron
						case MEDIUM:
							this.Medium <- electron
						case LOW:
							fallthrough
						default:
							// Unknown priority dump to low channel
							this.Low <- electron
						}
					}
				}
			}()
		} else {
			// TODO:
		}
	} else {
		// TODO:
	}

	return err
}

func (this Atomizer) Bond(electron Electron) (err error) {

	return err
}