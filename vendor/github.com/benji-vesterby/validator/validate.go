package validator

import "reflect"

// validator interface for checking to see if the struct has a Validate method assigned to it
type validator interface {
	Validate() (valid bool)
}

// IsValid reads in an object, and checks to see if the object implements the validator interface
// if the object does then it executes the objects validate method and returns that
func IsValid(obj interface{}) (valid bool) {

	// Using reflection pull the type associated with the object that is passed in. nil types are invalid.
	var tp reflect.Type
	if tp = reflect.TypeOf(obj); tp != nil {

		val := reflect.ValueOf(obj)

		// determine if the value is a pointer or not and whether it's nil if it is a pointer
		if val.Kind() != reflect.Ptr || (val.Kind() == reflect.Ptr && !val.IsNil()) {
			// TODO: Extend validate to read the struct parameters in and be able to use decoration to add more advanced automatic validation

			// Using a type assertion check to see if the object implements the validator interface
			// If it does then execute the validate method and assign that to the return value
			// Otherwise at this step the preliminary validation has occurred so return VALID
			if validate, ok := obj.(validator); ok {
				valid = validate.Validate()

			} else { // Valid because the object doesn't implement the validator interface, but is not nil
				valid = true
			}
		}
	}

	return valid
}

