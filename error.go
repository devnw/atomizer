package atomizer

import "github.com/benjivesterby/validator"

type atomError struct {
	err error
}

func (a atomError) Error() string {
	if !validator.Valid(a) {
		panic("invalid error")
	}

	return a.err.Error()
}

func (a atomError) String() string {
	if !validator.Valid(a) {
		panic("invalid error")
	}

	return a.err.Error()
}

func (a atomError) Unwrap() (err error) {

	err = a.err
	if aerr, ok := a.err.(atomError); ok {
		// Recursive unwrap to get the lowest error
		err = aerr.Unwrap()
	}

	return err
}

func (a atomError) Validate() (valid bool) {
	if a.err != nil {
		valid = true
	}

	return valid
}
