package conductors

import (
	"context"
	"encoding/json"
	"log"

	"github.com/benjivesterby/atomizer"
	"github.com/benjivesterby/validator"
	"github.com/streadway/amqp"
)

const (
	//DEFAULTADDRESS is the address to connect to rabbitmq
	DEFAULTADDRESS string = "amqp://guest:guest@localhost:5672/"
)

// Connect uses the connection string that is passed in to initialize
// the rabbitmq conductor
func Connect(connectionstring, exchange string) (atomizer.Conductor, error) {
	var err error
	mq := &rabbitmq{}

	if len(connectionstring) > 0 {
		// TODO: Add additional validation here for formatting later

		// Dial the connection
		if mq.conn, err = amqp.Dial(connectionstring); err == nil {
			if mq.channel, err = mq.conn.Channel(); err == nil {

				if mq.queue, err = mq.channel.QueueDeclare(
					"",    // name
					true,  // durable
					false, // delete when unused
					true,  // exclusive
					false, // no-wait
					nil,   // arguments
				); err == nil {

					if err = mq.channel.ExchangeDeclare(
						exchange, // name
						"fanout", // type
						true,     // durable
						false,    // auto-deleted
						false,    // internal
						false,    // no-wait
						nil,      // arguments

					); err == nil {

						if err = mq.channel.QueueBind(
							mq.queue.Name,
							"",
							exchange,
							false, //noWait -- TODO: see would this argument does
							nil,   //args
						); err == nil {

						}
					}
				}
			}
		}
	}

	return mq, err
}

type rabbitmq struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func (r *rabbitmq) ID() string {
	return "rabbitmq"
}

func (r *rabbitmq) Receive(ctx context.Context) <-chan []byte {
	var err error
	var in <-chan amqp.Delivery
	var out = make(chan []byte)

	if in, err = r.channel.Consume(

		r.queue.Name, //Queue
		"",           // consumer
		false,        // auto ack
		false,        // exclusive
		false,        // no local
		false,        // no wait
		nil,          // args
	); err == nil {
		go func(in <-chan amqp.Delivery, out chan<- []byte) {
			defer close(out)

			select {
			case <-ctx.Done():
				return
			case msg, ok := <-in:
				if ok {
					out <- msg.Body
					msg.Ack(false) //This acknolwedges a single delivery
				} else {
					return
				}
			}

		}(in, out)
	} else {
		close(out)
		// TODO: Handle error / panic
	}

	return out
}

func (r *rabbitmq) Complete(ctx context.Context, properties atomizer.Properties) (err error) {
	return err
}

func (r *rabbitmq) Send(ctx context.Context, electron atomizer.Electron) (result <-chan atomizer.Properties) {

	//TODO: need exchange name for UI

	result = make(chan atomizer.Properties)

	if validator.IsValid(electron) {
		go func(result chan atomizer.Properties) {

			var e []byte
			var err error
			
			

			if e, err = json.Marshal(electron); err == nil {

				if err = r.channel.Publish(
					"atomizer_topic", // exchange //TODO: should exchange be hard-coded?
					"electrons.id" + electron.ID,    // routing key //TODO: should routing key be hard coded?
					false,            // mandatory
					false,            // immediate
					amqp.Publishing{
						DeliveryMode: amqp.Persistent,
						ContentType:  "text/plain",
						Body:         e, //Send the elctron's properties
					}); err == nil {
					log.Printf("[x] Sent [%s]\n", string(e))
				}

				// Only kick off the electron for processing if there isn't already an
				// instance loaded in the system
	// 			if _, loaded := pt.results.LoadOrStore(electron.ID(), result); !loaded {

	// 				// Push the electron onto the input channel
	// 				select {
	// 				case <-ctx.Done():
	// 					return
	// 				case pt.input <- e:
	// 					// setup a monitoring thread for /basepath/electronid
	// 				}
	// 			} else {
	// 				defer close(result)
	// 				p := &properties{}
	// 				p.err = errors.Errorf("duplicate electron registration for EID [%s]", electron.ID())

	// 				result <- p
	// 			}
	// 		}
	// 	}(result)
	// }

	return result

}

func (r *rabbitmq) Close() {
	// TODO: set these up to be async, and sync.Once
	r.channel.Close()
	r.conn.Close()
}

// func failOnError(err error, msg string) {
// 	if err != nil {
// 		log.Fatalf("%s: %s", msg, err)
// 	}
// }

// func openChannel(conn *amqp.Connection) *amqp.Channel {

// 	ch, err := conn.Channel()
// 	failOnError(err, "Failed to open a channel")

// 	return ch

// }

// On spinup /path/for/electrons...
// Step one read from main path
// Step 2: when sending an electron monitor return path of basepath/electronid
