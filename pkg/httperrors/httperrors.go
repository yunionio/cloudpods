package httperrors

import (
	"fmt"
	"net/http"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

func HTTPError(w http.ResponseWriter, msg string, statusCode int, class string, error httputils.Error) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	body := jsonutils.NewDict()
	body.Add(jsonutils.NewInt(int64(statusCode)), "code")
	body.Add(jsonutils.NewString(msg), "details")
	body.Add(jsonutils.NewString(class), "class")
	err := jsonutils.NewDict()
	err.Add(jsonutils.NewString(error.Id), "id")
	err.Add(jsonutils.NewStringArray(error.Fields), "fields")
	body.Add(err, "data")
	w.Write([]byte(body.String()))
}

func JsonClientError(w http.ResponseWriter, e *httputils.JSONClientError) {
	HTTPError(w, e.Details, e.Code, e.Class, e.Data)
}

func GeneralServerError(w http.ResponseWriter, e error) {
	je, ok := e.(*httputils.JSONClientError)
	if ok {
		JsonClientError(w, je)
	} else {
		InternalServerError(w, fmt.Sprintf("%s", e))
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
