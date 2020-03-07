package conductors

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/benjivesterby/alog"
	"github.com/benjivesterby/atomizer"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

const (
	//DEFAULTADDRESS is the address to connect to rabbitmq
	DEFAULTADDRESS string = "amqp://guest:guest@localhost:5672/"
)

// Connect uses the connection string that is passed in to initialize
// the rabbitmq conductor
func Connect(ctx context.Context, connectionstring, inqueue string) (atomizer.Conductor, error) {
	var err error

	if ctx == nil {
		ctx = context.Background()
	}

	mq := &rabbitmq{
		ctx:               ctx,
		in:                inqueue,
		uuid:              uuid.New().String(),
		electronchans:     make(map[string]chan<- *atomizer.Properties),
		electronchanmutty: sync.Mutex{},
		pubs:              make(map[string]chan []byte),
		pubsmutty:         sync.Mutex{},
	}

	if len(connectionstring) > 0 {
		// TODO: Add additional validation here for formatting later

		// initialize the context of the conductor
		mq.ctx, mq.cancel = context.WithCancel(ctx)

		// Setup cleanup to run when the context closes
		go mq.Cleanup()

		// Dial the connection
		if mq.connection, err = amqp.Dial(connectionstring); err == nil {
			// TODO: connection established to rabbit
		} else {
			defer mq.cancel()
			err = errors.Errorf("error while opening connection to rabbitmq | %s", err.Error())
		}
	}

	return mq, err
}

//The rabbitmq struct uses the amqp library to connect to rabbitmq in order to send and receive
//from the message queue.
type rabbitmq struct {
	ctx    context.Context
	cancel context.CancelFunc

	// Incoming Requests
	in string

	// Queue for receiving results of sent messages
	uuid   string
	sender sync.Map

	electronchans     map[string]chan<- *atomizer.Properties
	electronchanmutty sync.Mutex
	once              sync.Once

	connection *amqp.Connection

	pubs      map[string]chan []byte
	pubsmutty sync.Mutex
}

func (r *rabbitmq) Cleanup() {
	<-r.ctx.Done()

	// TODO: should the error be ignored here?
	_ = r.connection.Close()
}

// Receive gets the atoms from the source that are available to atomize.
// Part of the Conductor interface
func (r *rabbitmq) Receive(ctx context.Context) <-chan *atomizer.Electron {
	electrons := make(chan *atomizer.Electron)

	go func(electrons chan<- *atomizer.Electron) {
		defer close(electrons)

		in := r.getReceiver(ctx, r.in)

		for {
			select {

			case <-ctx.Done():
				return
			case msg, ok := <-in:
				if ok {

					e := &atomizer.Electron{}
					if err := json.Unmarshal(msg, e); err == nil {
						r.sender.Store(e.ID, e.SenderID)

						select {
						case <-ctx.Done():
							return
						case electrons <- e:
							alog.Printf("electron [%s] received by conductor", e.ID)
						}
					} else {
						err = errors.Errorf("unable to parse electron %s", string(msg))
						alog.Errorf(err, "")
					}
				} else {
					return
				}
			}
		}
	}(electrons)

	return electrons
}

func (r *rabbitmq) fanResults(ctx context.Context) {
	results := r.getReceiver(ctx, r.uuid)

	alog.Printf("conductor [%s] receiver initialized", r.uuid)

	for {
		select {
		case <-ctx.Done():
			return
		case result, ok := <-results:
			if ok {

				go func(result []byte) {
					// Unwrap the object
					p := &atomizer.Properties{}
					if err := json.Unmarshal(result, p); err == nil {
						var c chan<- *atomizer.Properties

						// Pull the results channel for the electron
						r.electronchanmutty.Lock()
						c = r.electronchans[p.ElectronID]
						r.electronchanmutty.Unlock()

						// Ensure the channel is not nil
						if c != nil {

							// Close the channel after this result has been pushed
							defer close(c)

							select {
							case <-ctx.Done():
								return
							case c <- p: // push the result onto the channel
								alog.Printf("sent electron [%s] results to channel", p.ElectronID)
							}
						}
					} else {
						alog.Errorf(err, "error while un-marshalling results for conductor [%s]", r.uuid)
					}
				}(result)
			} else {
				select {
				case <-ctx.Done():
				default:
					panic("conductor results channel closed")
				}
			}
		}
	}
}

// Gets the list of messages that have been sent to the queue and returns them as a
// channel of byte arrays
func (r *rabbitmq) getReceiver(ctx context.Context, queue string) <-chan []byte {

	var err error
	var in <-chan amqp.Delivery
	var out = make(chan []byte)
	// Create the inbound processing exchanges and queues
	var c *amqp.Channel
	if c, err = r.connection.Channel(); err == nil {

		if _, err = c.QueueDeclare(
			queue, // name
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		); err == nil {

			// Prefetch variables
			if err = c.Qos(
				1,     // prefetch count
				0,     // prefetch size
				false, // global
			); err == nil {

				if in, err = c.Consume(

					queue, // Queue
					"",    // consumer
					true,  // auto ack
					false, // exclusive
					false, // no local
					false, // no wait
					nil,   // args
				); err == nil {
					go func(in <-chan amqp.Delivery, out chan<- []byte) {
						defer func() {
							_ = c.Close()
						}()

						defer close(out)

						for {
							select {
							case <-ctx.Done():
								return
							case msg, ok := <-in:
								if ok {
									out <- msg.Body
								} else {
									return
								}
							}
						}

					}(in, out)
				} else {
					close(out)
					// TODO: Handle error / panic
				}
			}
		}
	}

	return out
}

// Complete mark the completion of an electron instance with applicable statistics
func (r *rabbitmq) Complete(ctx context.Context, properties *atomizer.Properties) (err error) {

	if s, ok := r.sender.Load(properties.ElectronID); ok {

		if senderID, ok := s.(string); ok {

			var result []byte
			if result, err = json.Marshal(properties); err == nil {
				if err = r.publish(ctx, senderID, result); err == nil {
					alog.Printf("sent results for electron [%s] to sender [%s]", properties.ElectronID, senderID)
				} else {
					alog.Errorf(err, "error publishing results for electron [%s]", properties.ElectronID)
				}
			}
		}
	}

	return err
}

//Publishes an electron for processing or publishes a completed electron's properties
func (r *rabbitmq) publish(ctx context.Context, queue string, message []byte) (err error) {

	select {
	case <-ctx.Done():
		return
	case r.getPublisher(ctx, queue) <- message:
		// TODO:
	}

	return err
}

func (r *rabbitmq) getPublisher(ctx context.Context, queue string) chan<- []byte {
	r.pubsmutty.Lock()
	defer r.pubsmutty.Unlock()

	p := r.pubs[queue]

	// create the channel used for publishing and setup a go channel to monitor for publishing requests
	if p == nil {

		// Create the channel and update the map
		p = make(chan []byte)
		r.pubs[queue] = p

		// Create the new publisher and start the monitoring loop
		go func(ctx context.Context, connection *amqp.Connection, p <-chan []byte) {
			var c *amqp.Channel
			var err error
			if c, err = connection.Channel(); err == nil {
				defer func() {
					_ = c.Close()
				}()

				if _, err = c.QueueDeclare(
					queue, // name
					true,  // durable
					false, // delete when unused
					false, // exclusive
					false, // no-wait
					nil,   // arguments
				); err == nil {

					for {
						select {
						case <-ctx.Done():
							return
						case msg, ok := <-p:
							if ok {

								if err = c.Publish(
									"",    // exchange
									queue, // routing key
									false, // mandatory
									false, // immediate
									amqp.Publishing{
										ContentType: "application/json",
										Body:        msg,
									}); err == nil {

								} else {
									// TODO:
								}
							} else {
								return
							}
						}
					}
				} else {
					// TODO:
				}
			} else {
				// TODO:
			}
		}(ctx, r.connection, p)
	}

	return p
}

// Sends electrons back out through the conductor for additional processing
func (r *rabbitmq) Send(ctx context.Context, electron *atomizer.Electron) (<-chan *atomizer.Properties, error) {
	var e []byte
	var err error
	respond := make(chan *atomizer.Properties)

	// setup the results fan out
	go r.once.Do(func() { r.fanResults(ctx) })

	// TODO: Add in timeout here
	go func(ctx context.Context, electron *atomizer.Electron, respond chan<- *atomizer.Properties) {

		electron.SenderID = r.uuid

		if e, err = json.Marshal(electron); err == nil {
			var err error
			// ctx, cancel := context.WithTimeout(ctx, time.Second*30)
			// defer cancel()

			// Register the electron return channel prior to publishing the request
			r.electronchanmutty.Lock()
			r.electronchans[electron.ID] = respond
			r.electronchanmutty.Unlock()

			// publish the request to the message queue
			if err = r.publish(ctx, r.in, e); err == nil {
				alog.Printf("sent electron [%s] for processing\n", electron.ID)
			} else {
				alog.Errorf(err, "error sending electron [%s] for processing", electron.ID)
			}

		} else {
			alog.Errorf(err, "error while marshalling electron [%s]", electron.ID)
		}
	}(ctx, electron, respond)

	return respond, err
}

func (r *rabbitmq) Close() {

	// cancel out the internal context cleaning up the rabbit connection and channel
	r.cancel()
}
