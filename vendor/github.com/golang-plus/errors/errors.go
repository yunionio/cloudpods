package errors

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
)

// location represents a Location information of source code.
type location struct {
	File string
	Line int
}

// String returns the string of Location with filename:line format.
func (l *location) String() string {
	if len(l.File) == 0 {
		return ""
	}

	return fmt.Sprintf("%s:%d", l.File, l.Line)
}

// error2 represents an error.
type error2 struct {
	Inner    error
	Location *location
	Message  string
}

// Error implements builtin.error interface.
// It returns error message in a tree like format.
func (e *error2) Error() string {
	return TreeMessage(e)
}

func locate(depth int) *location {
	_, file, line, _ := runtime.Caller(depth + 1)
	return &location{
		File: file,
		Line: line,
	}
}

// Equal detects whether the error is equal to a given error.
// Errors are considered equal by this function if they are the same object, or if they both contain the same error inside.
func Equal(err1 error, err2 error) bool {
	if err1 == err2 {
		return true
	}

	if v, ok := err1.(*error2); ok {
		return Equal(v.Inner, err2)
	}

	if v, ok := err2.(*error2); ok {
		return Equal(v.Inner, err1)
	}

	return false
}

// TreeMessage returns the errors message in a tree like format:
// ├─ error message -> filename:line
// |  ├─ error message -> filename:line
// |     └─ error 3
func TreeMessage(err error) string {
	if err == nil {
		return ""
	}

	var buf bytes.Buffer
	depth := 1
	for {
		if depth > 1 {
			buf.WriteString("\n|")
			buf.WriteString(strings.Repeat(" ", (depth-1)*3-1))
		}

		if e, ok := err.(*error2); ok {
			buf.WriteString("├─ " + e.Message + " -> " + e.Location.String())
			err = e.Inner
		} else {
			buf.WriteString("└─ " + err.Error())
			err = nil
		}

		if err == nil {
			break
		}

		depth++
	}

	return buf.String()
}

// New returns a new Error.
// It's used to instead of built-in errors.New function.
func New(message string) error {
	return &error2{
		Location: locate(1),
		Message:  message,
	}
}

// Newf same as New, but with fmt.Printf-style parameters.
func Newf(format string, args ...interface{}) error {
	return &error2{
		Location: locate(1),
		Message:  fmt.Sprintf(format, args...),
	}
}

// Wrap returns a new wrapped Error.
func Wrap(err error, message string) error {
	return &error2{
		Inner:    err,
		Location: locate(2),
		Message:  message,
	}
}

// Wrapf same as Wrap, but with fmt.Printf-style parameters.
func Wrapf(err error, format string, args ...interface{}) error {
	return &error2{
		Inner:    err,
		Location: locate(2),
		Message:  fmt.Sprintf(format, args...),
	}
}
