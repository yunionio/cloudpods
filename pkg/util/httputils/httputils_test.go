package httputils

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
