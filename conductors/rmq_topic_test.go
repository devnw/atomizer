package conductors

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/streadway/amqp"
)

func TestReceiveMessage(t *testing.T) {

	//What the string is supposed to be
	ans := []byte("A test string")

	fmt.Println("before addMessage")

	addMessage(string(ans))

	fmt.Println("after addMessage")

	topic := CreateTopicReceiver()

	topic.DeclareExchange("atomizer_topic")

	topic.BindWithRoutingKey("#", "atomizer_topic")

	chanMsgs := topic.Receive(context.Background())

	for d := range chanMsgs {

		if string(d) == string(ans) {
			t.Errorf("Expected %v, got %v", "A test string", d)
		} else {
			fmt.Printf("Success")
		}

	}

}

func addMessage(message string) {

	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to RabbitMQ")
	defer conn.Close()

	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")
	defer ch.Close()

	err = ch.ExchangeDeclare(
		"atomizer_topic", // name
		"topic",          // type
		true,             // durable
		false,            // auto-deleted
		false,            // internal
		false,            // no-wait
		nil,              // arguments
	)

	body := message
	err = ch.Publish(
		"atomizer_topic", // exchange
		"isonomia",       // routing key
		false,            // mandatory
		false,            // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(body),
		})
	failOnError(err, "Failed to publish a message")

	log.Printf(" [x] Sent %s", body)

}
