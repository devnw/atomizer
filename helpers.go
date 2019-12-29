package atomizer

import (
	"fmt"
	"strings"
)

// ID returns the registration id for the passed in object type
func ID(v interface{}) string {
	return strings.Trim(fmt.Sprintf("%T", v), "*")
}
