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

	"yunion.io/x/onecloud/pkg/util/httputils"
)

func NewBadGatewayError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrBadGateway], string(ErrBadGateway), msg, params...)
}

func NewNotImplementedError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrNotImplemented], string(ErrNotImplemented), msg, params...)
}

func NewInternalServerError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrInternalError], string(ErrInternalError), msg, params...)
}

func NewResourceNotReadyError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrResourceNotReady], string(ErrResourceNotReady), msg, params...)
}

func NewOutOfResourceError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrOutOfResource], string(ErrOutOfResource), msg, params...)
}

func NewServerStatusError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrServerStatus], string(ErrServerStatus), msg, params...)
}

func NewPaymentError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrPayment], string(ErrPayment), msg, params...)
}

func NewImageNotFoundError(imageId string) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrImageNotFound], string(ErrImageNotFound), "Image %s not found", imageId)
}

func NewResourceNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrResourceNotFound], string(ErrResourceNotFound), msg, params...)
}

func NewResourceNotFoundError2(keyword, id string) *httputils.JSONClientError {
	return NewResourceNotFoundError("%s %s not found", keyword, id)
}

func NewSpecNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrSpecNotFound], string(ErrSpecNotFound), msg, params...)
}

func NewActionNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrActionNotFound], string(ErrActionNotFound), msg, params...)
}

func NewTenantNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrTenantNotFound], string(ErrTenantNotFound), msg, params...)
}

func NewUserNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrUserNotFound], string(ErrUserNotFound), msg, params...)
}

func NewInvalidStatusError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrInvalidStatus], string(ErrInvalidStatus), msg, params...)
}

func NewInputParameterError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrInputParameter], string(ErrInputParameter), msg, params...)
}

func NewWeakPasswordError() *httputils.JSONClientError {
	msg := ("password must be 12 chars of at least one digit, letter, uppercase letter and punctuate")
	return httputils.NewJsonClientError(httpErrorCode[ErrWeakPassword], string(ErrWeakPassword), msg)
}

func NewMissingParameterError(paramName string) *httputils.JSONClientError {
	msg := "Missing parameter %s"
	return httputils.NewJsonClientError(httpErrorCode[ErrMissingParameter], string(ErrMissingParameter), msg, paramName)
}

func NewPolicyDefinitionError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrPolicyDefinition], string(ErrPolicyDefinition), msg, params...)
}

func NewInsufficientResourceError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrInsufficientResource], string(ErrInsufficientResource), msg, params...)
}

func NewOutOfQuotaError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrOutOfQuota], string(ErrOutOfQuota), msg, params...)
}

func NewOutOfRangeError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrOutOfRange], string(ErrOutOfRange), msg, params...)
}

func NewOutOfLimitError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrOutOfLimit], string(ErrOutOfLimit), msg, params...)
}

func NewNotSufficientPrivilegeError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrNotSufficientPrivilege], string(ErrNotSufficientPrivilege), msg, params...)
}

func NewUnsupportOperationError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrUnsupportedOperation], string(ErrUnsupportedOperation), msg, params...)
}

func NewNotSupportedError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrNotSupported], string(ErrNotSupported), msg, params...)
}

func NewNotEmptyError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrNotEmpty], string(ErrNotEmpty), msg, params...)
}

func NewBadRequestError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrBadRequest], string(ErrBadRequest), msg, params...)
}

func NewUnauthorizedError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrUnauthorized], string(ErrUnauthorized), msg, params...)
}

func NewInvalidCredentialError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrInvalidCredential], string(ErrInvalidCredential), msg, params...)
}

func NewForbiddenError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrForbidden], string(ErrForbidden), msg, params...)
}

func NewNotFoundError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrNotFound], string(ErrNotFound), msg, params...)
}

func NewNotAcceptableError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrNotAcceptable], string(ErrNotAcceptable), msg, params...)
}

func NewDuplicateNameError(resName string, resId string) *httputils.JSONClientError {
	msg := "Duplicate name %s %s"
	return httputils.NewJsonClientError(httpErrorCode[ErrDuplicateName], string(ErrDuplicateName), msg, resName, resId)
}

func NewDuplicateIdError(resName string, resId string) *httputils.JSONClientError {
	msg := "Duplicate ID %s %s"
	return httputils.NewJsonClientError(httpErrorCode[ErrDuplicateId], string(ErrDuplicateId), msg, resName, resId)
}

func NewDuplicateResourceError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrDuplicateResource], string(ErrDuplicateResource), msg, params...)
}

func NewConflictError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrConflict], string(ErrConflict), msg, params...)
}

func NewResourceBusyError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrResourceBusy], string(ErrResourceBusy), msg, params...)
}

func NewRequireLicenseError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrRequireLicense], string(ErrRequireLicense), msg, params...)
}

func NewTimeoutError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrTimeout], string(ErrTimeout), msg, params...)
}

func NewProtectedResourceError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrProtectedResource], string(ErrProtectedResource), msg, params...)
}

func NewNoProjectError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrNoProject], string(ErrNoProject), msg, params...)
}

func NewServerError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[errors.ErrServer], string(errors.ErrServer), msg, params...)
}

func NewClientError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[errors.ErrClient], string(errors.ErrClient), msg, params...)
}

func NewUnclassifiedError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[errors.ErrUnclassified], string(errors.ErrUnclassified), msg, params...)
}

func NewTooLargeEntityError(msg string, params ...interface{}) *httputils.JSONClientError {
	return httputils.NewJsonClientError(httpErrorCode[ErrTooLarge], string(ErrTooLarge), msg, params...)
}
