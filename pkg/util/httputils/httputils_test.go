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

package httputils

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SErrorMsg struct {
	statusCode int
	Code       string
	Message    string
	Title      string
	Type       string
	ErrorCode  string
	ErrorMsg   string
	result     JSONClientErrorMsg
}

func TestError(t *testing.T) {

	for testName, msg := range map[string]SErrorMsg{
		"test1": {
			statusCode: 400,
			Code:       "VPC.0601",
			Message:    "Securitygroup id is invalid.",
			Title:      "",
			Type:       "",
			result: JSONClientErrorMsg{
				Error: &JSONClientError{
					Code:    400,
					Class:   "VPC.0601",
					Details: "Securitygroup id is invalid.",
				},
			},
		},
		"test2": {
			statusCode: 400,
			Code:       "3435",
			Message:    "Securitygroup id is invalid.",
			Title:      "",
			Type:       "",
			result: JSONClientErrorMsg{
				Error: &JSONClientError{
					Code:    3435,
					Class:   "",
					Details: "Securitygroup id is invalid.",
				},
			},
		},
		"test3": {
			statusCode: 400,
			ErrorCode:  "APIGW.0301",
			ErrorMsg:   "Incorrect IAM authentication information: verify aksk signature fail",
			result: JSONClientErrorMsg{
				Error: &JSONClientError{
					Code:    400,
					Class:   "APIGW.0301",
					Details: "Incorrect IAM authentication information: verify aksk signature fail",
				},
			},
		},
	} {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(msg.statusCode)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(jsonutils.Marshal(msg).String()))
		}))
		_, _, err := JSONRequest(ts.Client(), context.Background(), THttpMethod("GET"), ts.URL, nil, nil, true)
		if err != nil {
			respErr := JSONClientErrorMsg{}
			if err := json.Unmarshal([]byte(err.Error()), &respErr); err != nil {
				t.Error(err)
			}
			if msg.result.Error.Class != respErr.Error.Class {
				t.Errorf("expect %s class %s not %s", testName, msg.result.Error.Class, respErr.Error.Class)
			}
			if msg.result.Error.Code != respErr.Error.Code {
				t.Errorf("expect %s code %d not %d", testName, msg.result.Error.Code, respErr.Error.Code)
			}
			if msg.result.Error.Details != respErr.Error.Details {
				t.Errorf("expect %s detail %s not %s", testName, msg.result.Error.Class, respErr.Error.Class)
			}
		}

		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(msg.statusCode)
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(jsonutils.Marshal(map[string]SErrorMsg{"error": msg}).String()))
		}))
		_, _, err = JSONRequest(ts.Client(), context.Background(), THttpMethod("GET"), ts.URL, nil, nil, true)
		if err != nil {
			respErr := JSONClientErrorMsg{}
			if err := json.Unmarshal([]byte(err.Error()), &respErr); err != nil {
				t.Error(err)
			}
			if msg.result.Error.Class != respErr.Error.Class {
				t.Errorf("expect %s class %s not %s", testName, msg.result.Error.Class, respErr.Error.Class)
			}
			if msg.result.Error.Code != respErr.Error.Code {
				t.Errorf("expect %s code %d not %d", testName, msg.result.Error.Code, respErr.Error.Code)
			}
			if msg.result.Error.Details != respErr.Error.Details {
				t.Errorf("expect %s detail %s not %s", testName, msg.result.Error.Class, respErr.Error.Class)
			}
		}

	}
}

func TestErrorCause(t *testing.T) {
	err := errors.Error("TestError")
	jsonError := &JSONClientError{
		Code:    400,
		Class:   string(err),
		Details: "detailed test error",
	}
	wrapError := errors.Wrap(jsonError, "wrap1")
	if errors.Cause(wrapError) == err {
		t.Logf("%s", wrapError)
	} else {
		t.Errorf("wrapErro.Cause should be err: %#v != %#v", errors.Cause(wrapError), err)
	}
}
