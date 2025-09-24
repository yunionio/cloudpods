// Package jsonrpc provides JSON RPC utilities for serialisation of AWS
// requests and responses.
package jsonrpc

//go:generate go run ../../fixtures/protocol/generate.go ../../fixtures/protocol/input/json.json build_test.go
//go:generate go run ../../fixtures/protocol/generate.go ../../fixtures/protocol/output/json.json unmarshal_test.go

import (
	"encoding/json"
	"io/ioutil"
	"reflect"
	"strings"

	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/internal/apierr"
	"github.com/ks3sdklib/aws-sdk-go/internal/protocol/json/jsonutil"
)

var emptyJSON = []byte("{}")

// Build builds a JSON payload for a JSON RPC request.
func Build(req *aws.Request) {
	var buf []byte
	var err error
	if req.ParamsFilled() {
		v := reflect.ValueOf(req.Params).Elem()
		if field, ok := v.Type().FieldByName("SDKShapeTraits"); ok {
			if payloadName := field.Tag.Get("payload"); payloadName != "" {
				pfield, _ := v.Type().FieldByName(payloadName)
				if ptag := pfield.Tag.Get("type"); ptag == "" || ptag == "structure" {
					payload := reflect.Indirect(v.FieldByName(payloadName))
					if payload.IsValid() && payload.Interface() != nil {
						buf, err = jsonutil.BuildJSON(payload.Interface())
					}
				}
			}
		}
		if err != nil {
			req.Error = apierr.New("Marshal", "failed encoding JSON RPC request", err)
			return
		}
	} else {
		buf = emptyJSON
	}

	if req.Service.TargetPrefix != "" || string(buf) != "{}" {
		req.SetBufferBody(buf)
	}

	if req.Service.TargetPrefix != "" {
		target := req.Service.TargetPrefix + "." + req.Operation.Name
		req.HTTPRequest.Header.Add("X-Amz-Target", target)
	}
	if req.Service.JSONVersion != "" {
		jsonVersion := req.Service.JSONVersion
		req.HTTPRequest.Header.Add("Content-Type", "application/x-amz-json-"+jsonVersion)
	}
}

// Unmarshal unmarshals a response for a JSON RPC service.
func Unmarshal(req *aws.Request) {
	defer req.HTTPResponse.Body.Close()
	if req.DataFilled() {
		v := reflect.ValueOf(req.Data).Elem()
		if field, ok := v.Type().FieldByName("SDKShapeTraits"); ok {
			if payloadName := field.Tag.Get("payload"); payloadName != "" {
				pfield, _ := v.Type().FieldByName(payloadName)
				if ptag := pfield.Tag.Get("type"); ptag == "" || ptag == "structure" {
					payload := v.FieldByName(payloadName)
					if payload.IsValid() && payload.Interface() != nil {
						err := jsonutil.UnmarshalJSON(payload.Interface(), req.HTTPResponse.Body)
						if err != nil {
							req.Error = apierr.New("Unmarshal", "failed decoding JSON RPC response", err)
						}
					}
				}
			}
		}
	}
	return
}

// UnmarshalMeta unmarshals headers from a response for a JSON RPC service.
func UnmarshalMeta(req *aws.Request) {
	req.RequestID = req.HTTPResponse.Header.Get("x-amzn-requestid")
}

// UnmarshalError unmarshals an error response for a JSON RPC service.
func UnmarshalError(req *aws.Request) {
	defer req.HTTPResponse.Body.Close()
	bodyBytes, err := ioutil.ReadAll(req.HTTPResponse.Body)
	if err != nil {
		req.Error = apierr.New("Unmarshal", "failed reading JSON RPC error response", err)
		return
	}
	if len(bodyBytes) == 0 {
		req.Error = apierr.NewRequestError(
			apierr.New("Unmarshal", req.HTTPResponse.Status, nil),
			req.HTTPResponse.StatusCode,
			"",
		)
		return
	}
	var jsonErr jsonErrorResponse
	if err := json.Unmarshal(bodyBytes, &jsonErr); err != nil {
		req.Error = apierr.New("Unmarshal", "failed decoding JSON RPC error response", err)
		return
	}

	codes := strings.SplitN(jsonErr.Code, "#", 2)
	req.Error = apierr.NewRequestError(
		apierr.New(codes[len(codes)-1], jsonErr.Message, nil),
		req.HTTPResponse.StatusCode,
		"",
	)
}

type jsonErrorResponse struct {
	Code    string `json:"__type"`
	Message string `json:"message"`
}
