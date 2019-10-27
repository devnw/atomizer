package conductors

import (
	"context"

	"github.com/pkg/errors"
	"github.com/streadway/amqp"
)

//address to connect to rabbitmq
var defaultAddress string = "amqp://guest:guest@localhost:5672/"

var atomizerAddress string = "amqp://guest:my-rabbit:5672/"

//Type used to get a connection to RabbitMQ server and to receive messages in the form of routing keys
//Example of a routing key idea: "montecarlo.10000", where the first string is the atom and the second is a message
//specific to the atom
//Messages will be sent from the Atomizer website
type topic_receiver struct {
	conn    *amqp.Connection
	channel *amqp.Channel
	queue   amqp.Queue
}

func CreateTopicReceiver() topic_receiver {

	//will need to change this URL
	conn := connect(defaultAddress)
	ch := openChannel(conn)

	q, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when usused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue")

	return topic_receiver{conn: conn, channel: ch, queue: q}

}

func (topic *topic_receiver) DeclareExchange(exchangeName string) (err error) {

	if topic == nil {
		return errors.New("Topic receiver must not be nil")
	}

	err = topic.channel.ExchangeDeclare(
		exchangeName, // name
		"topic",      // type
		true,         // durable
		false,        // auto-deleted
		false,        // internal
		false,        // no-wait
		nil,          // arguments

	)
	failOnError(err, "Failed to declare an exchange")

	if err != nil {
		return err
	}
	return nil

}

func (topic *topic_receiver) CloseChannelAndConnection() {

	if topic.conn != nil {
		topic.conn.Close()
	}
	if topic.channel != nil {
		topic.channel.Close()
	}

}

func (topic_receiver *topic_receiver) BindWithRoutingKey(routingKey string, exchangeName string) (err error) {

	err = topic_receiver.channel.QueueBind(
		topic_receiver.queue.Name,
		routingKey,
		exchangeName,
		false, //noWait -- TODO: see would this argument does
		nil)   //args

	failOnError(err, "Failed to bind a queue")
	if err != nil {
		return err
	}
	return nil

}

//Default ID is rabbit_conductor
func (topic *topic_receiver) ID() string {
	return "rabbit_conductor"
}

// Receive gets the atoms from the source that are available to atomize

func (topic_receiver *topic_receiver) Receive(ctx context.Context) <-chan []byte {

	msgs, err := topic_receiver.channel.Consume(

		topic_receiver.queue.Name, //Queue
		"",                        // consumer
		true,                      // auto ack
		false,                     // exclusive
		false,                     // no local
		false,                     // no wait
		nil,                       // args

	)

	failOnError(err, "Failed to register a consumer")
	if err != nil {
		return nil
	}

	chanMsgs := make(chan []byte)
	go func() {

		for d := range msgs {
			chanMsgs <- d.Body
		}

	}()

	return chanMsgs

}

// Complete mark the completion of an electron instance with applicable statistics
// func (topic *topic_receiver) Complete(ctx context.Context, properties Properties) error {
// 	return nil
// }

// // Send sends electrons back out through the conductor for additional processing
// func (topic *topic_receiver) Send(ctx context.Context, electron Electron) (result <-chan Properties) {
// 	return nil
// }
