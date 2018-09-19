package httperrors

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/httputils"
)

func NewJsonClientError(code int, title string, msg string, error httputils.Error) *httputils.JSONClientError {
	err := httputils.JSONClientError{Code: code, Class: title, Details: msg, Data: error}
	return &err
}

func errorMessage(msg string, params ...interface{}) (string, httputils.Error) {
	fields := make([]string, len(params))
	for i, v := range params {
		fields[i] = fmt.Sprint(v)
	}

	error := httputils.Error{Id: msg, Fields: fields}
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}

	return msg, error
}

func NewBadGatewayError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(502, "BadGateway", msg, err)
}

func NewNotImplementedError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(501, "NotImplemented", msg, err)
}

func NewInternalServerError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(500, "InternalServerError", msg, err)
}

func NewResourceNotReadyError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(500, "ResourceNotReadyError", msg, err)
}

func NewOutOfResourceError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(500, "NewOutOfResourceError", msg, err)
}

func NewServerStatusError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(400, "ServerStatusError", msg, err)
}

func NewPaymentError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(402, "PaymentError", msg, err)
}

func NewImageNotFoundError(imageId string) *httputils.JSONClientError {
	msg, err := errorMessage("Image %s not found", imageId)
	return NewJsonClientError(404, "ImageNotFoundError", msg, err)
}

func NewResourceNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(404, "ResourceNotFoundError", msg, err)
}

func NewSpecNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(404, "SpecNotFoundError", msg, err)
}

func NewActionNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(404, "ActionNotFoundError", msg, err)
}

func NewTenantNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(404, "TenantNotFoundError", msg, err)
}

func NewUserNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(404, "UserNotFoundError", msg, err)
}

func NewInvalidStatusError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(400, "InvalidStatusError", msg, err)
}

func NewInputParameterError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(400, "InputParameterError", msg, err)
}

func NewWeakPasswordError() *httputils.JSONClientError {
	msg, err := errorMessage("password must be 12 chars of at least one digit, letter, uppercase letter and punctuate")
	return NewJsonClientError(400, "WeakPasswordError", msg, err)
}

func NewMissingParameterError(paramName string) *httputils.JSONClientError {
	msg, err := errorMessage("Missing parameter %s", paramName)
	return NewJsonClientError(400, "MissingParameterError", msg, err)
}

func NewInsufficientResourceError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(400, "InsufficientResourceError", msg, err)
}

func NewOutOfQuotaError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(400, "OutOfQuotaError", msg, err)
}

func NewNotSufficientPrivilegeError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(403, "NotSufficientPrivilegeError", msg, err)
}

func NewUnsupportOperationError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(406, "UnsupportOperationError", msg, err)
}

func NewNotEmptyError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(406, "NotEmptyError", msg, err)
}

func NewBadRequestError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(400, "BadRequestError", msg, err)
}

func NewUnauthorizedError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(401, "UnauthorizedError", msg, err)
}

func NewInvalidCredentialError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(401, "InvalidCredentialError", msg, err)
}

func NewForbiddenError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(403, "ForbiddenError", msg, err)
}

func NewNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(404, "NotFoundError", msg, err)
}

func NewNotAcceptableError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(406, "NotAcceptableError", msg, err)
}

func NewDuplicateNameError(resName string, resId string) *httputils.JSONClientError {
	msg, err := errorMessage("Duplicate %s %s", resName, resId)
	return NewJsonClientError(409, "DuplicateNameError", msg, err)
}

func NewConflictError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(409, "ConflictError", msg, err)
}

func NewResourceBusyError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(409, "ResourceBusyError", msg, err)
}

func NewRequireLicenseError(msg string, params ...interface{}) *httputils.JSONClientError {
	msg, err := errorMessage(msg, params...)
	return NewJsonClientError(402, "RequireLicenseError", msg, err)
}

func NewGeneralError(err error) *httputils.JSONClientError {
	switch err.(type) {
	case *httputils.JSONClientError:
		return err.(*httputils.JSONClientError)
	default:
		return NewInternalServerError(err.Error())
	}
}
