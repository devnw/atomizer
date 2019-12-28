package atomizer

import (
	"fmt"
	"strings"
)

// Id returns the registration id for the passed in object type
func Id(v interface{}) string {
	return strings.Trim(fmt.Sprintf("%T", v), "*")
}
