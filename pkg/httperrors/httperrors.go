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

	"golang.org/x/text/language"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/httputils"
)

type ctxLang uintptr

const ctxLangKey = ctxLang(0)
const LangHeader = "X-Yunion-Lang"

func SetLangHeader(ctx context.Context, header http.Header) bool {
	langv := ctx.Value(ctxLangKey)
	langTag, ok := langv.(language.Tag)
	if ok {
		header.Set(LangHeader, langTag.String())
	}
	return ok
}

func WithLangTag(ctx context.Context, tag language.Tag) context.Context {
	return context.WithValue(ctx, ctxLangKey, tag)
}

func WithLang(ctx context.Context, lang string) context.Context {
	tag, err := language.Parse(lang)
	if err != nil {
		tag = language.English
	}
	return WithLangTag(ctx, tag)
}

func WithRequestLang(ctx context.Context, req *http.Request) context.Context {
	if val := req.URL.Query().Get("lang"); val != "" {
		return WithLang(ctx, val)
	}
	if val := req.Header.Get(LangHeader); val != "" {
		return WithLang(ctx, val)
	}
	if cookie, err := req.Cookie("lang"); err == nil {
		return WithLang(ctx, cookie.Value)
	}
	return WithLangTag(ctx, language.English)
}

func SendHTTPErrorHeader(w http.ResponseWriter, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
}

func SetHTTPRedirectLocationHeader(w http.ResponseWriter, location string) {
	w.Header().Set("Location", location)
}

func HTTPError(ctx context.Context, w http.ResponseWriter, msg string, statusCode int, class string, err httputils.Error) {
	if statusCode >= 300 && statusCode <= 400 {
		SetHTTPRedirectLocationHeader(w, msg)
	}

	// 需要在调用w.WriteHeader方法之前，设置header才能生效
	SendHTTPErrorHeader(w, statusCode)

	var details string
	if err.Id == "" {
		details = msg
	} else {
		var (
			langv = ctx.Value(ctxLangKey)
			lang  language.Tag
		)
		if langv != nil {
			lang = langv.(language.Tag)
		} else {
			lang = language.English
		}
		a := make([]interface{}, len(err.Fields))
		for i := range err.Fields {
			a[i] = err.Fields[i]
		}
		details = P(lang, err.Id, a...)
	}
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewInt(int64(statusCode)), "code")
	body.Add(jsonutils.NewString(class), "class")
	body.Add(jsonutils.NewString(details), "details")
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
