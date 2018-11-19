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
	return newError(ERR_MISSING_KEY, "missing %q", key)
}

func newGeneralError(key string, err error) error {
	return newError(ERR_GENERAL, "general error for %q: %s", key, err)
}

func newInvalidTypeError(key string, typ string, err error) error {
	return newError(ERR_INVALID_TYPE, "expecting %s type for %q: %s", typ, key, err)
}

func newInvalidChoiceError(key string, choices Choices, choice string) error {
	return newError(ERR_INVALID_CHOICE, "invalid %q, want %s, got %s", key, choices, choice)
}

func newNotInRangeError(key string, value, lower, upper int64) error {
	return newError(ERR_NOT_IN_RANGE, "invalid %q: %d, want [%d,%d]", key, value, lower, upper)
}

func newInvalidValueError(key string, value string) error {
	return newError(ERR_INVALID_VALUE, "invalid %q: %s", key, value)
}

func newModelManagerError(modelKeyword string) error {
	return newError(ERR_MODEL_MANAGER, "failed getting model manager for %q", modelKeyword)
}

func newModelNotFoundError(modelKeyword, idOrName string, err error) error {
	errFmt := "cannot find %q with id/name %q"
	params := []interface{}{modelKeyword, idOrName}
	if err != sql.ErrNoRows {
		errFmt += ": %s"
		params = append(params, err.Error())
	}
	return newError(ERR_MODEL_NOT_FOUND, errFmt, params...)
}

func newError(typ ErrType, errFmt string, params ...interface{}) error {
	errFmt = fmt.Sprintf("%s: %s", typ, errFmt)
	if returnHttpError {
		switch typ {
		case ERR_SUCCESS:
			return nil
		case ERR_GENERAL, ERR_MODEL_MANAGER:
			return httperrors.NewInternalServerError(errFmt, params...)
		default:
			return httperrors.NewInputParameterError(errFmt, params...)
		}
	}
	err := &ValidateError{
		ErrType: typ,
		Msg:     fmt.Sprintf(errFmt, params...),
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
