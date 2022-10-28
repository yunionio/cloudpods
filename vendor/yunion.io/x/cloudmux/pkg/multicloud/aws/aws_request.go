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

package aws

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/corehandlers"
	"github.com/aws/aws-sdk-go/aws/request"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/private/protocol/query"
	xj "github.com/basgys/goxml2json"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

var UnmarshalHandler = request.NamedHandler{Name: "yunion.query.Unmarshal", Fn: Unmarshal}

func Unmarshal(r *request.Request) {
	defer r.HTTPResponse.Body.Close()
	if r.DataFilled() {
		var decoder *xml.Decoder
		if DEBUG {
			body, err := ioutil.ReadAll(r.HTTPResponse.Body)
			if err != nil {
				r.Error = awserr.NewRequestFailure(
					awserr.New("ioutil.ReadAll", "read response body", err),
					r.HTTPResponse.StatusCode,
					r.RequestID,
				)
				return
			}
			log.Debugf("response: \n%s", string(body))
			decoder = xml.NewDecoder(strings.NewReader(string(body)))
		} else {
			decoder = xml.NewDecoder(r.HTTPResponse.Body)
		}
		if r.ClientInfo.ServiceID == EC2_SERVICE_ID {
			err := decoder.Decode(r.Data)
			if err != nil {
				r.Error = awserr.NewRequestFailure(
					awserr.New("SerializationError", "failed decoding EC2 Query response", err),
					r.HTTPResponse.StatusCode,
					r.RequestID,
				)
			}
			return
		}
		for {
			tok, err := decoder.Token()
			if err != nil {
				if err == io.EOF {
					break
				}
				r.Error = awserr.NewRequestFailure(
					awserr.New("decoder.Token()", "get token", err),
					r.HTTPResponse.StatusCode,
					r.RequestID,
				)
				return
			}

			if tok == nil {
				break
			}

			switch typed := tok.(type) {
			case xml.CharData:
				continue
			case xml.StartElement:
				if typed.Name.Local == r.Operation.Name+"Result" {
					err = decoder.DecodeElement(r.Data, &typed)
					if err != nil {
						r.Error = awserr.NewRequestFailure(
							awserr.New("DecodeElement", "failed decoding Query response", err),
							r.HTTPResponse.StatusCode,
							r.RequestID,
						)
					}
					return
				}
			case xml.EndElement:
				break
			}
		}

	}
}

var buildHandler = request.NamedHandler{Name: "yunion.query.Build", Fn: Build}

func Build(r *request.Request) {
	body := url.Values{
		"Action":  {r.Operation.Name},
		"Version": {r.ClientInfo.APIVersion},
	}
	if r.Params != nil {
		if params, ok := r.Params.(map[string]string); ok {
			for k, v := range params {
				body.Add(k, v)
			}
		}
	}

	if DEBUG {
		log.Debugf("params: %s", body.Encode())
	}

	if !r.IsPresigned() {
		r.HTTPRequest.Method = "POST"
		r.HTTPRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
		r.SetBufferBody([]byte(body.Encode()))
	} else { // This is a pre-signed request
		r.HTTPRequest.Method = "GET"
		r.HTTPRequest.URL.RawQuery = body.Encode()
	}
}

var UnmarshalErrorHandler = request.NamedHandler{Name: "awssdk.ec2query.UnmarshalError", Fn: UnmarshalError}

type sAwsError struct {
	Errors struct {
		Type    string
		Code    string
		Message string
	} `json:"Error"`
	RequestID string
}

func (self sAwsError) Error() string {
	return jsonutils.Marshal(self).String()
}

func UnmarshalError(r *request.Request) {
	defer r.HTTPResponse.Body.Close()

	result, err := xj.Convert(r.HTTPResponse.Body)
	if err != nil {
		r.Error = errors.Wrapf(err, "xj.Convert")
		return
	}

	obj, err := jsonutils.Parse([]byte(result.String()))
	if err != nil {
		r.Error = errors.Wrapf(err, "jsonutils.Parse")
		return
	}

	respErr := &sAwsError{}
	if obj.Contains("ErrorResponse") {
		err = obj.Unmarshal(respErr, "ErrorResponse")
		if err != nil {
			r.Error = errors.Wrapf(err, "obj.Unmarshal")
			return
		}
	} else if obj.Contains("Response", "Errors") {
		err = obj.Unmarshal(respErr, "Response", "Errors")
		if err != nil {
			r.Error = errors.Wrapf(err, "obj.Unmarshal")
			return
		}
	}

	if strings.Contains(respErr.Errors.Code, "NotFound") || respErr.Errors.Code == "NoSuchEntity" {
		r.Error = errors.Wrapf(cloudprovider.ErrNotFound, jsonutils.Marshal(respErr).String())
		return
	}

	r.Error = respErr
	return
}

func (self *SAwsClient) request(regionId, serviceName, serviceId, apiVersion string, apiName string, params map[string]string, retval interface{}, assumeRole bool) error {
	if len(regionId) == 0 {
		regionId = self.getDefaultRegionId()
	}
	session, err := self.getAwsSession(regionId, assumeRole)
	if err != nil {
		return err
	}
	c := session.ClientConfig(serviceName)
	metadata := metadata.ClientInfo{
		ServiceName:   serviceName,
		ServiceID:     serviceId,
		SigningName:   c.SigningName,
		SigningRegion: c.SigningRegion,
		Endpoint:      c.Endpoint,
		APIVersion:    apiVersion,
	}

	if self.debug {
		logLevel := aws.LogLevelType(uint(aws.LogDebugWithRequestErrors) + uint(aws.LogDebugWithHTTPBody))
		c.Config.LogLevel = &logLevel
	}

	client := client.New(*c.Config, metadata, c.Handlers)
	client.Handlers.Sign.PushBackNamed(v4.SignRequestHandler)
	client.Handlers.Build.PushBackNamed(buildHandler)
	client.Handlers.Unmarshal.PushBackNamed(UnmarshalHandler)
	client.Handlers.UnmarshalMeta.PushBackNamed(query.UnmarshalMetaHandler)
	client.Handlers.UnmarshalError.PushBackNamed(UnmarshalErrorHandler)
	client.Handlers.Validate.Remove(corehandlers.ValidateEndpointHandler)
	return jsonRequest(client, apiName, params, retval, true)
}

func jsonRequest(cli *client.Client, apiName string, params map[string]string, retval interface{}, debug bool) error {
	op := &request.Operation{
		Name:       apiName,
		HTTPMethod: "POST",
		HTTPPath:   "/",
		Paginator: &request.Paginator{
			InputTokens:     []string{"NextToken"},
			OutputTokens:    []string{"NextToken"},
			LimitToken:      "MaxResults",
			TruncationToken: "",
		},
	}

	req := cli.NewRequest(op, params, retval)
	err := req.Send()
	if err != nil {
		if e, ok := err.(awserr.RequestFailure); ok && e.StatusCode() == 404 {
			return cloudprovider.ErrNotFound
		}
		return err
	}
	return nil
}
