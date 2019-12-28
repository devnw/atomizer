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
func Connect(connectionstring, inqueue string) (atomizer.Conductor, error) {
	var err error

	mq := &rabbitmq{
		in:                inqueue,
		uuid:              uuid.New().String(),
		electronchans:     make(map[string]chan<- *atomizer.Properties),
		electronchanmutty: sync.Mutex{},
	}

	if len(connectionstring) > 0 {
		// TODO: Add additional validation here for formatting later

		// Dial the connection
		mq.conn, err = amqp.Dial(connectionstring)
	}

	return mq, err
}

//The rabbitmq struct uses the amqp library to connect to rabbitmq in order to send and receive
//from the message queue.
type rabbitmq struct {
	conn *amqp.Connection

	// Incoming Requests
	in string

	// Queue for receiving results of sent messages
	uuid   string
	sender sync.Map

	electronchans     map[string]chan<- *atomizer.Properties
	electronchanmutty sync.Mutex
	once              sync.Once
}

func (r *rabbitmq) ID() string {
	return "rabbitmq"
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
	if c, err = r.conn.Channel(); err == nil {

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
						defer c.Close()
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
				r.publish(ctx, senderID, result)
				alog.Printf("sent results for electron [%s] to sender [%s]", properties.ElectronID, senderID)
			}
		}
	}

	return err
}

//Publishes an electron for processing or publishes a completed electron's properties
func (r *rabbitmq) publish(ctx context.Context, queue string, message []byte) (err error) {
	// Create the inbound processing exchanges and queues
	var results *amqp.Channel
	if results, err = r.conn.Channel(); err == nil {
		defer results.Close()
		if _, err = results.QueueDeclare(
			queue, // name
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		); err == nil {

			if err = results.Publish(
				"",    // exchange
				queue, // routing key
				false, // mandatory
				false, // immediate
				amqp.Publishing{
					ContentType: "application/json",
					Body:        message,
				}); err == nil {
			}
		}
	}

	return err
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
			}

			// // Create the inbound processing exchanges and queues
			// var atoms *amqp.Channel
			// if atoms, err = r.conn.Channel(); err == nil {
			// 	defer atoms.Close()

			// 	if _, err = atoms.QueueDeclare(
			// 		r.aqueue, // name
			// 		true,     // durable
			// 		false,    // delete when unused
			// 		false,    // exclusive
			// 		false,    // no-wait
			// 		nil,      // arguments
			// 	); err == nil {

			// 		// Create the listeners for results of processing that was pushed out
			// 		if err = atoms.Publish(
			// 			"",       // exchange
			// 			r.aqueue, // routing key
			// 			false,    // mandatory
			// 			false,    // immediate
			// 			amqp.Publishing{
			// 				DeliveryMode: amqp.Persistent,
			// 				ContentType:  "application/json",
			// 				Body:         e, //Send the electron's properties
			// 			}); err == nil {

			// 			var res <-chan []byte
			// 			if res, err = r.listen(ctx, electron.ID()); err == nil {
			// 				select {
			// 				case <-ctx.Done():
			// 					return
			// 				case r, ok := <-res:
			// 					if ok {
			// 						p := &atomizer.Properties{}
			// 						if err = json.Unmarshal(r, p); err == nil {
			// 							select {
			// 							case <-ctx.Done():
			// 								err = errors.Errorf("context cancelled for electron [%s]; timeout exceeded", electron.ID())
			// 								return
			// 							case respond <- p:
			// 								alog.Printf("sent electron [%s] result for completion\n", electron.ID())
			// 							}
			// 						}
			// 					} else {
			// 						return
			// 					}
			// 				}
			// 			}
			// 		}

			// 	}
			// }
		}
	}(ctx, electron, respond)

	return respond, err
}

// func (r *rabbitmq) listen(ctx context.Context, electronid string) (<-chan []byte, error) {
// 	results := make(chan []byte)
// 	var err error

// 	go func(ctx context.Context, electronid string, results chan<- []byte) {
// 		defer close(results)
// 		var in <-chan amqp.Delivery
// 		// Create the inbound processing exchanges and queues
// 		var atoms *amqp.Channel
// 		if atoms, err = r.conn.Channel(); err == nil {

// 			if _, err = atoms.QueueDeclare(
// 				r.resultex, // name
// 				true,       // durable
// 				false,      // delete when unused
// 				false,      // exclusive
// 				false,      // no-wait
// 				nil,        // arguments
// 			); err == nil {

// 				if in, err = atoms.Consume(

// 					r.resultex, // Queue
// 					"",         // consumer
// 					true,       // auto ack
// 					false,      // exclusive
// 					false,      // no local
// 					false,      // no wait
// 					nil,        // args
// 				); err == nil {

// 					select {
// 					case <-ctx.Done():
// 						return
// 					case res, ok := <-in:
// 						if ok {
// 							alog.Printf("received result for electron [%s]\n", electronid)
// 							results <- res.Body
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}(ctx, electronid, results)

// 	// go func(ctx context.Context, electronid string, results chan<- []byte) {
// 	// 	defer close(results)

// 	// 	var ch *amqp.Channel
// 	// 	if ch, err = r.conn.Channel(); err == nil {
// 	// 		defer ch.Close()

// 	// 		if err = ch.ExchangeDeclare(
// 	// 			r.resultex, // name
// 	// 			"topic",    // type
// 	// 			true,       // durable
// 	// 			false,      // auto-deleted
// 	// 			false,      // internal
// 	// 			false,      // no-wait
// 	// 			nil,        // arguments

// 	// 		); err == nil {

// 	// 			var q amqp.Queue
// 	// 			if q, err = ch.QueueDeclare(
// 	// 				"",    // name
// 	// 				true,  // durable
// 	// 				false, // delete when unused
// 	// 				true,  // exclusive
// 	// 				false, // no-wait
// 	// 				nil,   // arguments
// 	// 			); err == nil {

// 	// 				if err = ch.QueueBind(
// 	// 					q.Name,
// 	// 					electronid,
// 	// 					r.resultex,
// 	// 					false, //noWait -- TODO: see would this argument does
// 	// 					nil,   //args
// 	// 				); err == nil {

// 	// 					var msgs <-chan amqp.Delivery
// 	// 					if msgs, err = ch.Consume(
// 	// 						q.Name, // queue
// 	// 						"",     // consumer
// 	// 						true,   // auto ack
// 	// 						false,  // exclusive
// 	// 						false,  // no local
// 	// 						false,  // no wait
// 	// 						nil,    // args
// 	// 					); err == nil {
// 	// 						select {
// 	// 						case <-ctx.Done():
// 	// 							return
// 	// 						case res, ok := <-msgs:
// 	// 							if ok {
// 	// 								alog.Printf("received result for electron [%s]\n", electronid)
// 	// 								results <- res.Body
// 	// 							} else {
// 	// 								return
// 	// 							}
// 	// 						}
// 	// 					}
// 	// 				}
// 	// 			}
// 	// 		}
// 	// 	}
// 	// }(ctx, electronid, results)

// 	return results, err
// }

func (r *rabbitmq) Close() {
	r.conn.Close()
}
