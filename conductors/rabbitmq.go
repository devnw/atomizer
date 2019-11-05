package conductors

import (
	"context"

	"github.com/benjivesterby/alog"
	"github.com/benjivesterby/atomizer"
	"github.com/google/uuid"
	"github.com/streadway/amqp"
)

const (
	//DEFAULTADDRESS is the address to connect to rabbitmq
	DEFAULTADDRESS string = "amqp://guest:guest@localhost:5672/"
)

// Connect uses the connection string that is passed in to initialize
// the rabbitmq conductor
func Connect(connectionstring, exchange, route string) (atomizer.Conductor, error) {
	var err error
	mq := &rabbitmq{}

	if len(connectionstring) > 0 {
		// TODO: Add additional validation here for formatting later

		// Dial the connection
		if mq.conn, err = amqp.Dial(connectionstring); err == nil {
			if mq.channel, err = mq.conn.Channel(); err == nil {

				if mq.queue, err = mq.channel.QueueDeclare(
					uuid.New().String(), // name
					false,               // durable
					false,               // delete when unused
					true,                // exclusive
					false,               // no-wait
					nil,                 // arguments
				); err == nil {

					if err = mq.channel.ExchangeDeclare(
						exchange, // name
						"topic",  // type
						true,     // durable
						false,    // auto-deleted
						false,    // internal
						false,    // no-wait
						nil,      // arguments

					); err == nil {

						if err = mq.channel.QueueBind(
							mq.queue.Name,
							route,
							exchange,
							false, //noWait -- TODO: see would this argument does
							nil,   //args
						); err == nil {
							// TODO:
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
		true,         // auto ack
		false,        // exclusive
		false,        // no local
		false,        // no wait
		nil,          // args
	); err == nil {
		go func(in <-chan amqp.Delivery, out chan<- []byte) {

			for {
				select {
				case <-ctx.Done():
					defer close(out)
					return
				case msg, ok := <-in:
					if ok {
						alog.Println("pushing inbound message to consumer")
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

	return out
}

func (r *rabbitmq) Complete(ctx context.Context, properties atomizer.Properties) (err error) {
	return err
}

func (r *rabbitmq) Send(ctx context.Context, electron atomizer.Electron) (result <-chan atomizer.Properties) {
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
