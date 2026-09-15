// Package wraperr packages an error with another wrapped error
// This allows errors.Is to work without directly injecting the string of the wrapped error
package wraperr

// WrapErr wraps an underlying error with another error
//
// Example usage:
//
//	var ErrLimit = errors.New("Limit reached")
//	...
//	func someFunc() error {
//	  return wrapper.WrapErr{Err: fmt.Errorf("Limit %d reached on %s", limit, instance), Wrap: ErrLimit}
//	}
//	...
//	if errors.Is(err, ErrLimit) { ... }
//
// This makes it easier to include a custom error message while also allowing
type WrapErr struct {
	Err  error
	Wrap error
}

// New returns a new WrapErr
func New(err, wrap error) WrapErr {
	return WrapErr{Err: err, Wrap: wrap}
}

func (e WrapErr) Error() string {
	return e.Err.Error()
}

func (e WrapErr) Unwrap() error {
	return e.Wrap
}
