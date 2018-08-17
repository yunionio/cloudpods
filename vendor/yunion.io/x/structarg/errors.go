package structarg

import (
	"fmt"
)

type NotEnoughArgumentsError struct {
	argument Argument
}

func (e *NotEnoughArgumentsError) Error() string {
	return fmt.Sprintf("Not enough arguments, missing %s", e.argument)
}
