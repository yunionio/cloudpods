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

func NewGeneralError(err error) *httputils.JSONClientError {
	switch nerr := err.(type) {
	case *httputils.JSONClientError:
		return nerr
	case errors.Error:
		code, ok := httpErrorCode[nerr]
		if !ok {
			code = 500
		}
		return httputils.NewJsonClientError(code, string(nerr), err.Error())
	default:
		root := errors.Cause(err)
		switch nerr := root.(type) {
		case *httputils.JSONClientError:
			nerr.Details = err.Error()
			return nerr
		case errors.Error:
			code, ok := httpErrorCode[nerr]
			if !ok {
				code = 500
			}
			return httputils.NewJsonClientError(code, string(nerr), err.Error())
		default:
			return NewUnclassifiedError(err.Error())
		}
	}
}
