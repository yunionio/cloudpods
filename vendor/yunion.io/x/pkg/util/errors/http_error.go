package errors

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"gopkg.in/gin-gonic/gin.v1"
)

type HttpError struct {
	Status     int    `json:"code"`
	ErrMessage string `json:"details"`
}

func (e HttpError) Error() string {
	return fmt.Sprintf("HTTP Error %d: %s", e.Status, e.ErrMessage)
}

func NewHttpError(status int, err interface{}) error {
	err1, ok := err.(error)
	if ok {
		return HttpError{Status: status, ErrMessage: err1.Error()}
	}
	if reflect.ValueOf(err).Kind() == reflect.String {
		return HttpError{Status: status, ErrMessage: err.(string)}
	}
	return errors.New("NewHttpError received unknown params")
}

func HandleHttpError(c *gin.Context, err error) {
	err1, ok := err.(HttpError)
	if ok {
		if c.Request.Method == "HEAD" {
			c.AbortWithError(err1.Status, err1)
		} else {
			c.JSON(err1.Status, err1)
		}
	} else {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}
