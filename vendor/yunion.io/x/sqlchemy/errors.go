package sqlchemy

import (
	"errors"
)

var ErrNoDataToUpdate error
var ErrDuplicateEntry error

func init() {
	ErrNoDataToUpdate = errors.New("No data to update")
	ErrDuplicateEntry = errors.New("duplicate entry")
}
