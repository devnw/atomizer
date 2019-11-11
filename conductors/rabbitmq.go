package conductors

import (
	"context"
	"encoding/json"

	"github.com/benjivesterby/alog"
	"github.com/benjivesterby/atomizer"
	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

const (
	//DEFAULTADDRESS is the address to connect to rabbitmq
	DEFAULTADDRESS string = "amqp://guest:guest@localhost:5672/"
)

// Connect uses the connection string that is passed in to initialize
// the rabbitmq conductor
func Connect(connectionstring, aqueue, resultex string) (atomizer.Conductor, error) {
	var err error
	mq := &rabbitmq{aqueue: aqueue, resultex: resultex}

	if len(connectionstring) > 0 {
		// TODO: Add additional validation here for formatting later

		// Dial the connection
		if mq.conn, err = amqp.Dial(connectionstring); err == nil {

			// Create the inbound processing exchanges and queues
			if mq.atoms, err = mq.conn.Channel(); err == nil {

				if _, err = mq.atoms.QueueDeclare(
					aqueue, // name
					true,   // durable
					false,  // delete when unused
					false,  // exclusive
					false,  // no-wait
					nil,    // arguments
				); err == nil {

				}
			}

			// Create the listeners for results of processing that was pushed out
		}
	}

	return mq, err
}

type rabbitmq struct {
	conn *amqp.Connection

	// Incoming Requests
	aqueue string
	atoms  *amqp.Channel

	// Finished results
	resultex string
}

func (r *rabbitmq) ID() string {
	return "rabbitmq"
}

func (r *rabbitmq) Receive(ctx context.Context) <-chan []byte {
	var err error
	var in <-chan amqp.Delivery
	var out = make(chan []byte)

	// Prefetch variables
	if err = r.atoms.Qos(
		1,     // prefetch count
		0,     // prefetch size
		false, // global
	); err == nil {

		if in, err = r.atoms.Consume(

			r.aqueue, // Queue
			"",       // consumer
			true,     // auto ack
			false,    // exclusive
			false,    // no local
			false,    // no wait
			nil,      // args
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
	}

	return out
}

func (r *rabbitmq) Complete(ctx context.Context, properties *atomizer.Properties) (err error) {

	var ch *amqp.Channel
	if ch, err = r.conn.Channel(); err == nil {
		defer ch.Close()

		if err = ch.ExchangeDeclare(
			r.resultex, // name
			"topic",    // type
			true,       // durable
			false,      // auto-deleted
			false,      // internal
			false,      // no-wait
			nil,        // arguments

		); err == nil {
			var result []byte
			if result, err = json.Marshal(properties); err == nil {

				if err = ch.Publish(
					r.resultex,            // exchange
					properties.ElectronID, // routing key
					false,                 // mandatory
					false,                 // immediate
					amqp.Publishing{
						ContentType: "application/json",
						Body:        result,
					}); err == nil {

					alog.Printf("Electron [%s] complete, pushed results to conductor\n", properties.ElectronID)
				}
			}
		}
	}

	return err
}

func (r *rabbitmq) Send(ctx context.Context, electron atomizer.Electron) (err error) {
	var e []byte

	if e, err = json.Marshal(electron); err == nil {

		if err = r.atoms.Publish(
			"",       // exchange
			r.aqueue, // routing key
			false,    // mandatory
			false,    // immediate
			amqp.Publishing{
				DeliveryMode: amqp.Persistent,
				ContentType:  "application/json",
				Body:         e, //Send the electron's properties
			}); err == nil {
			alog.Printf("sent electron [%s] for processing\n", electron.ID())

			respond := electron.Respond()
			if respond != nil {

				// TODO: Add in timeout here
				go func(respond chan<- *atomizer.Properties) {
					var res []byte
					if res, err = r.listen(ctx, electron.ID()); err == nil {
						p := &atomizer.Properties{}
						if err = json.Unmarshal(res, p); err == nil {
							select {
							case <-ctx.Done():
								return
							case respond <- p:
								alog.Printf("sent electron [%s] result for completion\n", electron.ID())
							}
						}
					}
				}(respond)
			}
		}
	}

	return err
}

func (r *rabbitmq) listen(ctx context.Context, electronid string) (results []byte, err error) {

	var ch *amqp.Channel
	if ch, err = r.conn.Channel(); err == nil {
		defer ch.Close()

		if err = ch.ExchangeDeclare(
			r.resultex, // name
			"topic",    // type
			true,       // durable
			false,      // auto-deleted
			false,      // internal
			false,      // no-wait
			nil,        // arguments

		); err == nil {

			var q amqp.Queue
			if q, err = ch.QueueDeclare(
				"",    // name
				true,  // durable
				false, // delete when unused
				true,  // exclusive
				false, // no-wait
				nil,   // arguments
			); err == nil {

				if err = ch.QueueBind(
					q.Name,
					electronid,
					r.resultex,
					false, //noWait -- TODO: see would this argument does
					nil,   //args
				); err == nil {

					var msgs <-chan amqp.Delivery
					if msgs, err = ch.Consume(
						q.Name, // queue
						"",     // consumer
						true,   // auto ack
						false,  // exclusive
						false,  // no local
						false,  // no wait
						nil,    // args
					); err == nil {
						select {
						case <-ctx.Done():
							return nil, nil
						case res, ok := <-msgs:
							if ok {
								alog.Printf("received result for electron [%s]\n", electronid)
								results = res.Body
							} else {
								return nil, errors.New("channel closed without result")
							}
						}
					}
				}
			}
		}
	}

	return results, err
}

func (r *rabbitmq) Close() {
	// TODO: set these up to be async, and sync.Once
	r.atoms.Close()
	r.conn.Close()
}
