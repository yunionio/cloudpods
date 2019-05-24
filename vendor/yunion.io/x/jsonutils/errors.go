package jsonutils

import (
	"fmt"

	"github.com/pkg/errors"
)

var (
	ErrJsonDictFailInsert = errors.New("fail to insert object")

	ErrInvalidJsonDict    = errors.New("not a valid JSONDict")
	ErrInvalidJsonArray   = errors.New("not a valid JSONArray")
	ErrInvalidJsonInt     = errors.New("not a valid number")
	ErrInvalidJsonFloat   = errors.New("not a valid float")
	ErrInvalidJsonBoolean = errors.New("not a valid boolean")
	ErrInvalidJsonString  = errors.New("not a valid string")

	ErrJsonDictKeyNotFound = errors.New("key not found")

	ErrUnsupported     = errors.New("unsupported operation")
	ErrOutOfKeyRange   = errors.New("out of key range")
	ErrOutOfIndexRange = errors.New("out of index range")

	ErrInvalidChar = errors.New("invalid char")
	ErrInvalidHex  = errors.New("invalid hex")
	ErrInvalidRune = errors.New("invalid 4 byte rune")

	ErrTypeMismatch         = errors.New("unmarshal type mismatch")
	ErrArrayLengthMismatch  = errors.New("unmarshal array length mismatch")
	ErrInterfaceUnsupported = errors.New("do not known how to deserialize json into this interface type")
	ErrMapKeyMustString     = errors.New("map key must be string")

	ErrMisingInputField = errors.New("missing input field")
	ErrNilInputField    = errors.New("nil input field")

	ErrYamlMissingDictKey = errors.New("Cannot find JSONDict key")
	ErrYamlIllFormat      = errors.New("Illformat")
)

type JSONError struct {
	pos    int
	substr string
	msg    string
}

func (e *JSONError) Error() string {
	return fmt.Sprintf("JSON error %s at %d: %s...", e.msg, e.pos, e.substr)
}

func NewJSONError(str []byte, pos int, msg string) *JSONError {
	sublen := 10
	start := pos - sublen
	end := pos + sublen
	if start < 0 {
		start = 0
	}
	if end > len(str) {
		end = len(str)
	}
	substr := append(str[start:pos], '^')
	substr = append(substr, str[pos:end]...)
	return &JSONError{pos: pos, substr: string(substr), msg: msg}
}
