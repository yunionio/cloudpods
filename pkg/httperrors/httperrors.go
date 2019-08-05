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
	"net/http"
	"runtime/debug"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/httputils"
)

func SendHTTPErrorHeader(w http.ResponseWriter, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
}

func SetHTTPRedirectLocationHeader(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
}

func HTTPError(w http.ResponseWriter, msg string, statusCode int, class string, error httputils.Error) {
	if statusCode >= 300 && statusCode <= 400 {
		SetHTTPRedirectLocationHeader(w, msg)
	}

	// 需要在调用w.WriteHeader方法之前，设置header才能生效
	SendHTTPErrorHeader(w, statusCode)

	body := jsonutils.NewDict()
	body.Add(jsonutils.NewInt(int64(statusCode)), "code")
	body.Add(jsonutils.NewString(msg), "details")
	body.Add(jsonutils.NewString(class), "class")
	err := jsonutils.NewDict()
	err.Add(jsonutils.NewString(error.Id), "id")
	err.Add(jsonutils.NewStringArray(error.Fields), "fields")
	body.Add(err, "data")
	w.Write([]byte(body.String()))
	log.Errorf("Send error %s", err)
	if statusCode >= 500 {
		debug.PrintStack()
	}
}

func JsonClientError(w http.ResponseWriter, e *httputils.JSONClientError) {
	HTTPError(w, e.Details, e.Code, e.Class, e.Data)
}

func GeneralServerError(w http.ResponseWriter, e error) {
	je, ok := e.(*httputils.JSONClientError)
	if ok {
		JsonClientError(w, je)
	} else {
		InternalServerError(w, "%s", e)
	}
}

func BadRequestError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewBadRequestError(msg, params...))
}

func PaymentError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewPaymentError(msg, params...))
}

func UnauthorizedError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewUnauthorizedError(msg, params...))
}

func InvalidCredentialError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewInvalidCredentialError(msg, params...))
}

func ForbiddenError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewForbiddenError(msg, params...))
}

func NotFoundError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewNotFoundError(msg, params...))
}

func NotImplementedError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewNotImplementedError(msg, params...))
}

func NotAcceptableError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewNotAcceptableError(msg, params...))
}

func InvalidInputError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewInputParameterError(msg, params...))
}

func InputParameterError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewInputParameterError(msg, params...))
}

func MissingParameterError(w http.ResponseWriter, param string) {
	JsonClientError(w, NewMissingParameterError(param))
}

func ConflictError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewConflictError(msg, params...))
}

func InternalServerError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewInternalServerError(msg, params...))
}

func BadGatewayError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewBadGatewayError(msg, params...))
}

func TenantNotFoundError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewTenantNotFoundError(msg, params...))
}

func OutOfQuotaError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewOutOfQuotaError(msg, params...))
}

func NotSufficientPrivilegeError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewNotSufficientPrivilegeError(msg, params...))
}

func ResourceNotFoundError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewResourceNotFoundError(msg, params...))
}

func TimeoutError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewTimeoutError(msg, params...))
}

func ProtectedResourceError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewProtectedResourceError(msg, params...))
}

func NoProjectError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewNoProjectError(msg, params...))
}
