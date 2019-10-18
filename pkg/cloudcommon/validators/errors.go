// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validators

import (
	"database/sql"
	"fmt"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/util/choices"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

var returnHttpError = true

type ErrType uintptr

const (
	ERR_SUCCESS ErrType = iota
	ERR_GENERAL         // uncategorized error
	ERR_MISSING_KEY
	ERR_INVALID_TYPE
	ERR_INVALID_CHOICE
	ERR_INVALID_LENGTH
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
	ERR_INVALID_LENGTH:  "Invalid length error",
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

func newInvalidChoiceError(key string, choices choices.Choices, choice string) error {
	return newError(ERR_INVALID_CHOICE, "invalid %q, want %s, got %s", key, choices, choice)
}

func newStringTooShortError(key string, got, want int) error {
	return newError(ERR_INVALID_LENGTH, "%q too short, got %d, min %s", key, got, want)
}

func newStringTooLongError(key string, got, want int) error {
	return newError(ERR_INVALID_LENGTH, "%q too long, got %d, max %s", key, got, want)
}

func newNotInRangeError(key string, value, lower, upper int64) error {
	return newError(ERR_NOT_IN_RANGE, "invalid %q: %d, want [%d,%d]", key, value, lower, upper)
}

func newInvalidValueError(key string, value string) error {
	return newError(ERR_INVALID_VALUE, "invalid %q: %s", key, value)
}

func newInvalidValueErrorEx(key string, err error) error {
	return newError(ERR_INVALID_VALUE, "invalid %q: %v", key, err)
}

func newInvalidStructError(key string, err error) error {
	errFmt := "invalid %q: "
	params := []interface{}{key}
	jsonClientErr, ok := err.(*httputils.JSONClientError)
	if ok {
		errFmt += httperrors.MsgTmplToFmt(jsonClientErr.Data.Id)
		for _, f := range jsonClientErr.Data.Fields {
			params = append(params, f)
		}
	}
	return newError(ERR_INVALID_VALUE, errFmt, params...)
}

func newModelManagerError(modelKeyword string) error {
	return newError(ERR_MODEL_MANAGER, "failed getting model manager for %q", modelKeyword)
}

func newModelNotFoundError(modelKeyword, idOrName string, err error) error {
	errFmt := "cannot find %q with id/name %q"
	params := []interface{}{modelKeyword, idOrName}
	if err != nil && err != sql.ErrNoRows {
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
		case ERR_MODEL_NOT_FOUND:
			return httperrors.NewResourceNotFoundError(errFmt, params...)
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
