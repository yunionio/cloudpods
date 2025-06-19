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
	"context"
	"net/http"
	"runtime/debug"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/i18n"
)

var (
	timeZone *time.Location
)

func init() {
	timeZone = time.Local
}

func GetTimeZone() *time.Location {
	return timeZone
}

func SetTimeZone(tzStr string) {
	if tz, _ := time.LoadLocation(tzStr); tz != nil {
		timeZone = tz
	}
}

func SendHTTPErrorHeader(w http.ResponseWriter, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
}

func SetHTTPRedirectLocationHeader(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
}

type Error struct {
	Code    int    `json:"code,omitzero"`
	Class   string `json:"class,omitempty"`
	Details string `json:"details,omitempty"`
}

func NewErrorFromJCError(ctx context.Context, je *httputils.JSONClientError) Error {
	err := Error{
		Code:    je.Code,
		Class:   je.Class,
		Details: formatDetails(ctx, je.Data, je.Details),
	}
	return err
}

func NewErrorFromGeneralError(ctx context.Context, e error) Error {
	je := NewGeneralError(e)
	return NewErrorFromJCError(ctx, je)
}

func formatDetails(ctx context.Context, errData httputils.Error, msg string) string {
	var details string
	if errData.Id == "" {
		details = msg
	} else {
		lang := appctx.Lang(ctx)
		a := make([]interface{}, len(errData.Fields))
		for i := range errData.Fields {
			a[i] = errData.Fields[i]
		}
		details = i18n.P(lang, errData.Id, a...)
	}
	return details
}

func HTTPError(ctx context.Context, w http.ResponseWriter, msg string, statusCode int, class string, errData httputils.Error) {
	details := formatDetails(ctx, errData, msg)
	if statusCode >= 300 && statusCode <= 400 {
		SetHTTPRedirectLocationHeader(w, details)
	}

	// 需要在调用w.WriteHeader方法之前，设置header才能生效
	SendHTTPErrorHeader(w, statusCode)

	err := Error{
		Code:    statusCode,
		Class:   class,
		Details: details,
	}
	body := jsonutils.Marshal(err).(*jsonutils.JSONDict)
	body.Set("time", jsonutils.NewString(time.Now().In(timeZone).Format(time.RFC3339)))
	w.Write([]byte(body.String()))
	log.Errorf("Send error %s", details)
	if statusCode >= 500 {
		debug.PrintStack()
	}
}

func JsonClientError(ctx context.Context, w http.ResponseWriter, e *httputils.JSONClientError) {
	HTTPError(ctx, w, e.Details, e.Code, e.Class, e.Data)
}

func GeneralServerError(ctx context.Context, w http.ResponseWriter, e error) {
	je := NewGeneralError(e)
	JsonClientError(ctx, w, je)
}

func BadRequestError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewBadRequestError(msg, params...))
}

func PaymentError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewPaymentError(msg, params...))
}

func UnauthorizedError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewUnauthorizedError(msg, params...))
}

func InvalidCredentialError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewInvalidCredentialError(msg, params...))
}

func ForbiddenError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewForbiddenError(msg, params...))
}

func NotFoundError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewNotFoundError(msg, params...))
}

func NotImplementedError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewNotImplementedError(msg, params...))
}

func NotAcceptableError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewNotAcceptableError(msg, params...))
}

func InvalidInputError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewInputParameterError(msg, params...))
}

func InputParameterError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewInputParameterError(msg, params...))
}

func MissingParameterError(ctx context.Context, w http.ResponseWriter, param string) {
	JsonClientError(ctx, w, NewMissingParameterError(param))
}

func ConflictError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewConflictError(msg, params...))
}

func InternalServerError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewInternalServerError(msg, params...))
}

func BadGatewayError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewBadGatewayError(msg, params...))
}

func TenantNotFoundError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewTenantNotFoundError(msg, params...))
}

func OutOfQuotaError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewOutOfQuotaError(msg, params...))
}

func NotSufficientPrivilegeError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewNotSufficientPrivilegeError(msg, params...))
}

func ResourceNotFoundError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewResourceNotFoundError(msg, params...))
}

func TimeoutError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewTimeoutError(msg, params...))
}

func ProtectedResourceError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewProtectedResourceError(msg, params...))
}

func NoProjectError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewNoProjectError(msg, params...))
}

func TooManyRequestsError(ctx context.Context, w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(ctx, w, NewTooManyRequestsError(msg, params...))
}
