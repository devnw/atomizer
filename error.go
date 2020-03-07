package atomizer

import (
	"fmt"
)

type aErr struct {
	err error
	msg string
}

func (a aErr) Error() string {
	return a.String()
}

func (a aErr) String() (s string) {

	s = a.msg
	if a.err != nil {
		s = fmt.Sprintf("%s - [ %s ]", s, a.err.Error())
	}

	return s
}

func (a aErr) Unwrap() (err error) {

	err = a.err
	if aerr, ok := a.err.(aErr); ok {
		// Recursive unwrap to get the lowest error
		err = aerr.Unwrap()
	}

	return err
}

func (a aErr) Validate() (valid bool) {
	if len(a.msg) > 0 {
		valid = true
	}

	return valid
}
