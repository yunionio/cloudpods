package validators

import (
	"database/sql"
	"fmt"

	"yunion.io/x/onecloud/pkg/httperrors"
)

var returnHttpError = true

type ErrType uintptr

const (
	ERR_SUCCESS ErrType = iota
	ERR_GENERAL         // uncategorized error
	ERR_MISSING_KEY
	ERR_INVALID_TYPE
	ERR_INVALID_CHOICE
	ERR_NOT_IN_RANGE
	ERR_INVALID_VALUE
	ERR_MODEL_MANAGER
	ERR_MODEL_NOT_FOUND
)

var errTypeToString = map[ErrType]string{
	ERR_SUCCESS:         "No error",
	ERR_GENERAL:         "General error",
	ERR_MISSING_KEY:     "Missing key error",
	ERR_INVALID_TYPE:    "Invalid type error",
	ERR_INVALID_CHOICE:  "Invalid choice error",
	ERR_NOT_IN_RANGE:    "Not in range error",
	ERR_INVALID_VALUE:   "Invalid value error",
	ERR_MODEL_MANAGER:   "Model manager error",
	ERR_MODEL_NOT_FOUND: "Model not found error",
}

func (errType ErrType) String() string {
	s, ok := errTypeToString[errType]
	if ok {
		return s
	}
	return "unknown error"
}

type ValidateError struct {
	ErrType ErrType
	Msg     string
}

func (ve *ValidateError) Error() string {
	return ve.Msg
}

// TODO let each validator provide the error
func newMissingKeyError(key string) error {
	msg := fmt.Sprintf("missing %q", key)
	return newError(ERR_MISSING_KEY, msg)
}

func newGeneralError(key string, err error) error {
	msg := fmt.Sprintf("general error for %q: %s", key, err)
	return newError(ERR_GENERAL, msg)
}

func newInvalidTypeError(key string, typ string, err error) error {
	msg := fmt.Sprintf("expecting %s type for %q: %s", typ, key, err)
	return newError(ERR_INVALID_TYPE, msg)
}

func newInvalidChoiceError(key string, choices Choices, choice string) error {
	msg := fmt.Sprintf("invalid %q, want %s, got %s", key, choices, choice)
	return newError(ERR_INVALID_CHOICE, msg)
}

func newNotInRangeError(key string, value, lower, upper int64) error {
	msg := fmt.Sprintf("invalid %q: %d, want [%d,%d]", key, value, lower, upper)
	return newError(ERR_NOT_IN_RANGE, msg)
}

func newInvalidValueError(key string, value string) error {
	msg := fmt.Sprintf("invalid %q: %s", key, value)
	return newError(ERR_INVALID_VALUE, msg)
}

func newModelManagerError(modelKeyword string) error {
	msg := fmt.Sprintf("internal error: getting model manager for %q failed",
		modelKeyword)
	return newError(ERR_MODEL_MANAGER, msg)
}

func newModelNotFoundError(modelKeyword, idOrName string, err error) error {
	msg := fmt.Sprintf("cannot find %q with id/name %q",
		modelKeyword, idOrName)
	if err != sql.ErrNoRows {
		msg += ": " + err.Error()
	}
	return newError(ERR_MODEL_NOT_FOUND, msg)
}

func newError(typ ErrType, msg string) error {
	err := &ValidateError{
		ErrType: typ,
		Msg:     msg,
	}
	if returnHttpError {
		switch typ {
		case ERR_SUCCESS:
			return nil
		case ERR_GENERAL, ERR_MODEL_MANAGER:
			return httperrors.NewInternalServerError(msg)
		default:
			return httperrors.NewInputParameterError(msg)
		}
	}
	return err
}

func IsModelNotFoundError(err error) bool {
	ve, ok := err.(*ValidateError)
	if ok && ve.ErrType == ERR_MODEL_NOT_FOUND {
		return true
	}
	return false
}
