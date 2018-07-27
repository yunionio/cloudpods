package httperrors

import (
	"fmt"

	"github.com/yunionio/pkg/util/httputils"
)

func NewJsonClientError(code int, title string, msg string) *httputils.JSONClientError {
	err := httputils.JSONClientError{Code: code, Class: title, Details: msg}
	return &err
}

func NewBadGatewayError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(502, "BadGateway", msg)
}

func NewNotImplementedError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(501, "NotImplemented", msg)
}

func NewInternalServerError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(500, "InternalServerError", msg)
}

func NewResourceNotReadyError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(500, "ResourceNotReadyError", msg)
}

func NewServerStatusError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(400, "ServerStatusError", msg)
}

func NewPaymentError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(402, "PaymentError", msg)
}

func NewImageNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(404, "ImageNotFoundError", msg)
}

func NewResourceNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(404, "ResourceNotFoundError", msg)
}

func NewSpecNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(404, "SpecNotFoundError", msg)
}

func NewActionNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(404, "ActionNotFoundError", msg)
}

func NewTenantNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(404, "TenantNotFoundError", msg)
}

func NewUserNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(404, "UserNotFoundError", msg)
}

func NewInvalidStatusError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(400, "InvalidStatusError", msg)
}

func NewInputParameterError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(400, "InputParameterError", msg)
}

func NewInsufficientResourceError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(400, "InsufficientResourceError", msg)
}

func NewOutOfQuotaError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(400, "OutOfQuotaError", msg)
}

func NewNotSufficientPrivilegeError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(403, "NotSufficientPrivilegeError", msg)
}

func NewUnsupportOperationError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(406, "UnsupportOperationError", msg)
}

func NewNotEmptyError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(406, "NotEmptyError", msg)
}

func NewBadRequestError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(400, "BadRequestError", msg)
}

func NewUnauthorizedError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(401, "UnauthorizedError", msg)
}

func NewInvalidCredentialError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(401, "InvalidCredentialError", msg)
}

func NewForbiddenError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(403, "ForbiddenError", msg)
}

func NewNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(404, "NotFoundError", msg)
}

func NewNotAcceptableError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(406, "NotAcceptableError", msg)
}

func NewDuplicateNameError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(409, "DuplicateNameError", msg)
}

func NewConflictError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(409, "ConflictError", msg)
}

func NewResourceBusyError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(409, "ResourceBusyError", msg)
}

func NewRequireLicenseError(msg string, params ...interface{}) *httputils.JSONClientError {
	if len(params) > 0 {
		msg = fmt.Sprintf(msg, params...)
	}
	return NewJsonClientError(402, "RequireLicenseError", msg)
}

func NewGeneralError(err error) *httputils.JSONClientError {
	switch err.(type) {
	case *httputils.JSONClientError:
		return err.(*httputils.JSONClientError)
	default:
		return NewInternalServerError(err.Error())
	}
}
