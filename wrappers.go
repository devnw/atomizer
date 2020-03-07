package atomizer

import (
	"context"

	"github.com/pkg/errors"
)

func wrapcancel(c context.CancelFunc) (err error) {

	if c != nil {
		c()
	} else {
		err = errors.New("cancel function is nil")
	}

	return err
}
