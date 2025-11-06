package log

import "errors"

type errorWithLevel struct {
	Level Level
	error
}

func (me errorWithLevel) Unwrap() error {
	return me.error
}

// Extracts the most recent error level added to err with [WithLevel], or NotSet.
func ErrorLevel(err error) Level {
	var withLevel errorWithLevel
	if !errors.As(err, &withLevel) {
		return NotSet
	}
	return withLevel.Level
}

// Adds the error level to err, it can be extracted with [ErrorLevel].
func WithLevel(level Level, err error) error {
	return errorWithLevel{level, err}
}
