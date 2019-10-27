package conductors

import (
	"log"

	"github.com/streadway/amqp"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func connect(address string) *amqp.Connection {

	//TODO: will need the correct URL
	conn, err := amqp.Dial(address)

	failOnError(err, "Failed to connect to RabbitMQ")

	return conn

}

func openChannel(conn *amqp.Connection) *amqp.Channel {

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")

	return ch

}

// On spinup /path/for/electrons...
// Step one read from main path
// Step 2: when sending an electron monitor return path of basepath/electronid
