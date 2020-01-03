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
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"yunion.io/x/jsonutils"
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

type ResponseHeaderTimeoutHandler struct{}

func (h *ResponseHeaderTimeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	time.Sleep(12 * time.Second)
	w.Write([]byte("hello"))
}

func TestResponseHeaderTimeout(t *testing.T) {
	rand.Seed(time.Now().Unix())
	port := rand.Intn(1000) + 53000
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        &ResponseHeaderTimeoutHandler{},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		s.ListenAndServe()
	}()

	clientWait(t)
	cli := GetClient(true, 0)
	resp, err := cli.Get(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err == nil {
		t.Errorf("Read shoud error %s", err)
	} else {
		t.Logf("Read error %s", err)
	}
	CloseResponse(resp)
	s.Close()
}

type idleTimeoutHandler struct{}

func (h *idleTimeoutHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(200)
	for i := 0; i < 4096; i++ {
		for j := 0; j < 4096; j++ {
			w.Write([]byte("<html>ABCDEFD</html>\n"))
		}
		time.Sleep(1 * time.Second)
	}
}

func clientWait(t *testing.T) {
	t.Logf("wait 2 seconds for server ...")
	time.Sleep(2 * time.Second)
	t.Logf("Start client ...")
}

func TestIdleTimeout(t *testing.T) {
	rand.Seed(time.Now().Unix())
	port := rand.Intn(1000) + 53000
	t.Logf("Start serve at %d", port)
	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", port),
		Handler:        &idleTimeoutHandler{},
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   300 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	go func() {
		s.ListenAndServe()
	}()

	clientWait(t)
	cli := GetClient(true, 15*time.Second)
	resp, err := cli.Get(fmt.Sprintf("http://127.0.0.1:%d", port))
	if err != nil {
		t.Errorf("Read error %s", err)
	}
	buf := make([]byte, 4096)
	offset := 0
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			offset += n
		}
		if err != nil {
			t.Logf("read error %s", err)
			break
		}
	}
	t.Logf("read offset %d", offset)
	CloseResponse(resp)
	s.Close()
}
