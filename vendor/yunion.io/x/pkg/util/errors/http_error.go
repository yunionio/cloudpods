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

package errors

import (
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/gin-gonic/gin"
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
