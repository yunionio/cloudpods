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

package httperrors

import (
	"yunion.io/x/pkg/errors"
)

const (
	ErrBadGateway       = errors.Error("BadGateway")
	ErrNotImplemented   = errors.ErrNotImplemented
	ErrInternalError    = errors.Error("InternalServerError")
	ErrResourceNotReady = errors.Error("ResourceNotReadyError")
	ErrPayment          = errors.Error("PaymentError")

	ErrImageNotFound    = errors.Error("ImageNotFoundError")
	ErrResourceNotFound = errors.Error("ResourceNotFoundError")
	ErrSpecNotFound     = errors.Error("SpecNotFoundError")
	ErrActionNotFound   = errors.Error("ActionNotFoundError")
	ErrTenantNotFound   = errors.Error("TenantNotFoundError")

	ErrServerStatus     = errors.Error("ServerStatusError")
	ErrInvalidStatus    = errors.ErrInvalidStatus
	ErrInvalidIdpStatus = errors.Error("InvalidIdpStatus")
	ErrInvalidFormat    = errors.ErrInvalidFormat

	ErrInputParameter   = errors.Error("InputParameterError")
	ErrWeakPassword     = errors.Error("WeakPasswordError")
	ErrMissingParameter = errors.Error("MissingParameterError")

	ErrInsufficientResource = errors.Error("InsufficientResourceError")
	ErrOutOfResource        = errors.Error("OutOfResource")
	ErrOutOfQuota           = errors.Error("OutOfQuotaError")
	ErrOutOfRange           = errors.Error("OutOfRange")
	ErrOutOfLimit           = errors.Error("OutOfLimit")

	ErrNotSufficientPrivilege = errors.Error("NotSufficientPrivilegeError")

	ErrUnsupportedOperation = errors.Error("UnsupportOperationError")
	ErrNotSupported         = errors.ErrNotSupported

	ErrNotEmpty     = errors.Error("NotEmptyError")
	ErrBadRequest   = errors.Error("BadRequestError")
	ErrEmptyRequest = errors.Error("EmptyRequestError")

	ErrUnauthorized      = errors.Error("UnauthorizedError")
	ErrInvalidCredential = errors.Error("InvalidCredentialError")
	ErrUnauthenticated   = ErrInvalidCredential
	ErrForbidden         = errors.Error("ForbiddenError")

	ErrNotFound = errors.ErrNotFound

	ErrNotAcceptable = errors.Error("NotAcceptableError")

	ErrDuplicateName     = errors.Error("DuplicateNameError")
	ErrDuplicateResource = errors.Error("DuplicateResourceError")
	ErrConflict          = errors.Error("ConflictError")
	ErrDuplicateId       = errors.ErrDuplicateId

	ErrResourceBusy   = errors.Error("ResourceBusyError")
	ErrRequireLicense = errors.Error("RequireLicenseError")

	ErrTimeout           = errors.ErrTimeout
	ErrProtectedResource = errors.Error("ProtectedResourceError")
	ErrNoProject         = errors.Error("NoProjectError")

	ErrInvalidProvider     = errors.Error("InvalidProvider")
	ErrNoBalancePermission = errors.Error("NoBalancePermission")

	ErrTooLarge = errors.Error("TooLargeEntity")

	ErrTooManyAttempts = errors.Error("TooManyFailedAttempts")
	ErrTooManyRequests = errors.Error("TooManyRequests")

	ErrUnsupportedProtocol = errors.ErrUnsupportedProtocol

	ErrPolicyDefinition = errors.Error("PolicyDefinitionError")

	ErrUserNotFound                = errors.Error("UserNotFound")
	ErrUserLocked                  = errors.Error("UserLocked")
	ErrUserDisabled                = errors.Error("UserDisabled")
	ErrWrongPassword               = errors.Error("WrongPassword")
	ErrIncorrectUsernameOrPassword = errors.Error("IncorrectUsernameOrPassword")

	ErrServiceAbnormal = errors.Error("ServiceAbnormal")

	ErrInvalidAccessKey = errors.Error("InvalidAccessKey")
	ErrNoPermission     = errors.Error("NoPermission")
)

var (
	httpErrorCode = map[errors.Error]int{
		errors.ErrClient:       400,
		errors.ErrServer:       500,
		errors.ErrUnclassified: 500,

		ErrInvalidProvider:     400,
		ErrNoBalancePermission: 403,

		ErrBadGateway:       502,
		ErrNotImplemented:   501,
		ErrInternalError:    500,
		ErrResourceNotReady: 500,
		ErrPayment:          402,

		ErrImageNotFound:    404,
		ErrResourceNotFound: 404,
		ErrSpecNotFound:     404,
		ErrActionNotFound:   404,
		ErrTenantNotFound:   404,
		ErrUserNotFound:     404,

		ErrServerStatus:  400,
		ErrInvalidStatus: 400,
		ErrInvalidFormat: 400,

		ErrInvalidIdpStatus: 400,

		ErrInputParameter:   400,
		ErrWeakPassword:     400,
		ErrMissingParameter: 400,

		ErrInsufficientResource: 400,
		ErrOutOfResource:        500,
		ErrOutOfQuota:           400,
		ErrOutOfRange:           400,
		ErrOutOfLimit:           400,

		ErrNotSufficientPrivilege: 403,

		ErrUnsupportedOperation: 406,
		ErrNotSupported:         406,

		ErrNotEmpty:     406,
		ErrBadRequest:   400,
		ErrEmptyRequest: 400,

		ErrUnauthorized:      401,
		ErrInvalidCredential: 401,
		ErrForbidden:         403,

		ErrNotFound: 404,

		ErrNotAcceptable: 406,

		ErrDuplicateName:     409,
		ErrDuplicateResource: 409,
		ErrConflict:          409,
		ErrDuplicateId:       409,

		ErrResourceBusy: 409,

		ErrRequireLicense: 402,

		ErrTimeout:           504,
		ErrProtectedResource: 403,
		ErrNoProject:         403,

		ErrTooLarge: 413,

		ErrTooManyAttempts: 429,
		ErrTooManyRequests: 429,

		ErrUserLocked:   423,
		ErrUserDisabled: 423,

		ErrWrongPassword:               401,
		ErrIncorrectUsernameOrPassword: 401,

		ErrPolicyDefinition: 409,

		ErrInvalidAccessKey: 400,

		ErrServiceAbnormal: 499,
	}
)

func RegisterErrorHttpCode(err errors.Error, code int) {
	if _, ok := httpErrorCode[err]; ok {
		panic("Error has been registered: " + string(err))
	}
	httpErrorCode[err] = code
}
