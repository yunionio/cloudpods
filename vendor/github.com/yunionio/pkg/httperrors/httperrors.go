package httperrors

import (
	"fmt"
	"net/http"

	"github.com/yunionio/jsonutils"
	"github.com/yunionio/pkg/util/httputils"
)

func HTTPError(w http.ResponseWriter, msg string, statusCode int, class string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewInt(int64(statusCode)), "code")
	body.Add(jsonutils.NewString(msg), "details")
	body.Add(jsonutils.NewString(class), "class")
	w.Write([]byte(body.String()))
}

func JsonClientError(w http.ResponseWriter, e *httputils.JSONClientError) {
	HTTPError(w, e.Details, e.Code, e.Class)
}

func GeneralServerError(w http.ResponseWriter, e error) {
	je, ok := e.(*httputils.JSONClientError)
	if ok {
		JsonClientError(w, je)
	} else {
		InternalServerError(w, fmt.Sprintf("%s", e))
	}
}

func BadRequestError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewBadRequestError(msg))
}

func PaymentError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewPaymentError(msg))
}

func UnauthorizedError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewUnauthorizedError(msg))
}

func InvalidCredentialError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewInvalidCredentialError(msg))
}

func ForbiddenError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewForbiddenError(msg))
}

func NotFoundError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewNotFoundError(msg))
}

func NotImplementedError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewNotImplementedError(msg))
}

func NotAcceptableError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewNotAcceptableError(msg))
}

func InvalidInputError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewInputParameterError(msg))
}

func ConflictError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewConflictError(msg))
}

func InternalServerError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewInternalServerError(msg))
}

func BadGatewayError(w http.ResponseWriter, msg string) {
	JsonClientError(w, NewBadGatewayError(msg))
}

func TenantNotFoundError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewTenantNotFoundError(msg, params...))
}

func OutOfQuotaError(w http.ResponseWriter, msg string, params ...interface{}) {
	JsonClientError(w, NewOutOfQuotaError(msg, params...))
}
