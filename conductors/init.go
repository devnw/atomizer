package conductors

import (
	"context"
)

func init() {
	Register(context.Background(), "rabbit", &rabbitmq{})
}
