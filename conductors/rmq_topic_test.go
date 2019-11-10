package conductors

import (
	"context"
	"testing"

	"github.com/benjivesterby/atomizer"
	"github.com/streadway/amqp"
)

func TestReceiveMessage(t *testing.T) {
	var err error

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//What the string is supposed to be
	ans := []byte("A test string")

	var c atomizer.Conductor
	if c, err = Connect(DEFAULTADDRESS, "atomizer_topic"); err == nil {

		if err = addMessage(t, string(ans)); err == nil {

			msgs := c.Receive(ctx)

			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-msgs:
					if ok {

						if string(msg) == string(ans) {
							t.Log("Success")
						} else {
							t.Errorf("Expected %v, got %v\n", string(ans), string(msg))
						}
					} else {
						return
					}
				}
			}
		}
	}

	// topic := CreateTopicReceiver()

	// topic.DeclareExchange("atomizer_topic")

	// topic.BindWithRoutingKey("#", "atomizer_topic")

	// for d := range chanMsgs {

	// 	if string(d) == string(ans) {
	// 		t.Errorf("Expected %v, got %v", "A test string", d)
	// 	} else {
	// 		fmt.Printf("Success")
	// 	}

	// }

}

//Error could be from the connection getting closed after the functions exits
//Exchange deleted
func addMessage(t *testing.T, message string) (err error) {

	var conn *amqp.Connection
	if conn, err = amqp.Dial(DEFAULTADDRESS); err == nil {
		defer conn.Close()

		var ch *amqp.Channel
		if ch, err = conn.Channel(); err == nil {
			defer ch.Close()

			if err = ch.ExchangeDeclare(
				"atomizer_topic", // name
				"fanout",         // type
				true,             // durable
				false,            // auto-deleted
				false,            // internal
				false,            // no-wait
				nil,              // arguments
			); err == nil {
				body := message

				if err = ch.Publish(
					"atomizer_topic", // exchange
					"",               // routing key
					false,            // mandatory
					false,            // immediate
					amqp.Publishing{
						DeliveryMode: amqp.Persistent,
						ContentType:  "text/plain",
						Body:         []byte(body),
					}); err == nil {
					t.Logf("[x] Sent [%s]\n", body)
				}
			}
		}
	}

	return err
}
