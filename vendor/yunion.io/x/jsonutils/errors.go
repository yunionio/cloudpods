package jsonutils

import (
	"fmt"

	"yunion.io/x/pkg/errors"
)

const (
	ErrJsonDictFailInsert = errors.Error("fail to insert object")

	ErrInvalidJsonDict    = errors.Error("not a valid JSONDict")
	ErrInvalidJsonArray   = errors.Error("not a valid JSONArray")
	ErrInvalidJsonInt     = errors.Error("not a valid number")
	ErrInvalidJsonFloat   = errors.Error("not a valid float")
	ErrInvalidJsonBoolean = errors.Error("not a valid boolean")
	ErrInvalidJsonString  = errors.Error("not a valid string")

	ErrJsonDictKeyNotFound = errors.Error("key not found")

	ErrUnsupported     = errors.Error("unsupported operation")
	ErrOutOfKeyRange   = errors.Error("out of key range")
	ErrOutOfIndexRange = errors.Error("out of index range")

	ErrInvalidChar = errors.Error("invalid char")
	ErrInvalidHex  = errors.Error("invalid hex")
	ErrInvalidRune = errors.Error("invalid 4 byte rune")

	ErrTypeMismatch         = errors.Error("unmarshal type mismatch")
	ErrArrayLengthMismatch  = errors.Error("unmarshal array length mismatch")
	ErrInterfaceUnsupported = errors.Error("do not known how to deserialize json into this interface type")
	ErrMapKeyMustString     = errors.Error("map key must be string")

	ErrMissingInputField = errors.Error("missing input field")
	ErrNilInputField     = errors.Error("nil input field")

	ErrYamlMissingDictKey = errors.Error("Cannot find JSONDict key")
	ErrYamlIllFormat      = errors.Error("Illformat")
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
