package atomizer

import "context"

func wrapcancel(c context.CancelFunc) error {

	if c == nil {
		return simple("cancel function is nil", nil)
	}

	c()

	return nil
}
