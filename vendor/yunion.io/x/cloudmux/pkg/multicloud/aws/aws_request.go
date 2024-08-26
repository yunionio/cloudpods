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
	"fmt"
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
		if r.Config.LogLevel != nil && r.Config.LogLevel.AtLeast(aws.LogDebugWithHTTPBody) {
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
		if r.ClientInfo.ServiceID == EC2_SERVICE_ID || r.ClientInfo.ServiceID == CDN_SERVICE_ID {
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
				if typed.Name.Local == r.Operation.Name+"Result" || (typed.Name.Local == r.Operation.Name+"Response" && r.ClientInfo.ServiceID == ROUTE53_SERVICE_ID) {
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
	body := url.Values{}
	if !strings.HasPrefix(r.HTTPRequest.URL.Host, "route53") {
		body.Set("Action", r.Operation.Name)
		body.Set("Version", r.ClientInfo.APIVersion)
	}
	var params map[string]string = map[string]string{}
	if r.Params != nil {
		var ok bool
		params, ok = r.Params.(map[string]string)
		if ok {
			for k, v := range params {
				body.Add(k, v)
			}
		}
	}

	if r.Config.LogLevel != nil && r.Config.LogLevel.AtLeast(aws.LogDebugWithHTTPBody) && r.ClientInfo.ServiceID != ROUTE53_SERVICE_ID {
		log.Debugf("params: %s", body.Encode())
	}

	if r.ClientInfo.ServiceID == CDN_SERVICE_ID {
		switch r.HTTPRequest.Method {
		case "GET":
			r.HTTPRequest.URL.RawQuery = body.Encode()
			return
		}
	}
	if r.ClientInfo.ServiceID == ROUTE53_SERVICE_ID {
		switch r.HTTPRequest.Method {
		case "GET":
			r.HTTPRequest.URL.RawQuery = body.Encode()
			return
		case "POST":
			body, err := xml.MarshalIndent(AwsXmlRequest{
				params: params,
				Local:  r.Operation.Name + "Request",
				Spec:   "https://route53.amazonaws.com/doc/2013-04-01/",
			}, "", "  ")
			if err != nil {
				r.Error = errors.Wrapf(err, "Marshal xml request")
				return
			}
			if r.Config.LogLevel != nil && r.Config.LogLevel.AtLeast(aws.LogDebugWithHTTPBody) {
				log.Debugf("params: %s", string(body))
			}
			r.SetBufferBody(body)
		}
		return
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

	defer func() {
		if r.Error != nil {
			log.Errorf("request: %s %s error: %v", r.HTTPRequest.URL.String(), r.Params, r.Error)
		}
	}()

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

	if strings.Contains(respErr.Errors.Code, "NotFound") || strings.HasPrefix(respErr.Errors.Code, "NoSuch") {
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

	client := client.New(*c.Config.WithMaxRetries(0), metadata, c.Handlers)
	client.Handlers.Sign.PushBackNamed(v4.SignRequestHandler)
	client.Handlers.Build.PushBackNamed(buildHandler)
	client.Handlers.Unmarshal.PushBackNamed(UnmarshalHandler)
	client.Handlers.UnmarshalMeta.PushBackNamed(query.UnmarshalMetaHandler)
	client.Handlers.UnmarshalError.PushBackNamed(UnmarshalErrorHandler)
	client.Handlers.Validate.Remove(corehandlers.ValidateEndpointHandler)
	return jsonRequest(client, apiName, params, retval)
}

func jsonRequest(cli *client.Client, apiName string, params map[string]string, retval interface{}) error {
	method, path := "POST", "/"
	if cli.ServiceID == CDN_SERVICE_ID {
		for _, prefix := range []string{"List", "Get"} {
			if strings.HasPrefix(apiName, prefix) {
				method = "GET"
			}
		}
		for k, v := range map[string]string{
			"ListDistributions2020_05_31": "distribution",
			"GetDistribution2020_05_31":   "distribution",
		} {
			if apiName == k {
				path = fmt.Sprintf("/2020-05-31/%s", v)
				if id, ok := params["Id"]; ok {
					path = fmt.Sprintf("/2020-05-31/%s/%s", v, strings.TrimPrefix(id, "/"))
					delete(params, "Id")
				}
			}
		}
	}

	if cli.ServiceID == ROUTE53_SERVICE_ID {
		for _, prefix := range []string{"List", "Get"} {
			if strings.HasPrefix(apiName, prefix) {
				method = "GET"
			}
		}
		if strings.HasPrefix(apiName, "Delete") {
			method = "DELETE"
		}
		// https://github.com/aws/aws-sdk-go/blob/main/service/route53/api.go
		for k, v := range map[string]string{
			"ListHostedZones":            "hostedzone",
			"ListGeoLocations":           "geolocations",
			"CreateHostedZone":           "hostedzone",
			"GetTrafficPolicyInstance":   "trafficpolicyinstance",
			"GetTrafficPolicy":           "trafficpolicy",
			"GetHostedZone":              "",
			"DeleteHostedZone":           "",
			"AssociateVPCWithHostedZone": "",
			"ListResourceRecordSets":     "",
			"ChangeResourceRecordSets":   "",
		} {
			if apiName == k {
				path = fmt.Sprintf("/2013-04-01/%s", v)
				if id, ok := params["Id"]; ok {
					path = fmt.Sprintf("/2013-04-01/%s", strings.TrimPrefix(id, "/"))
					suffix, _ := map[string]string{
						"AssociateVPCWithHostedZone":    "associatevpc",
						"DisassociateVPCFromHostedZone": "disassociatevpc",
						"ListResourceRecordSets":        "rrset",
						"ChangeResourceRecordSets":      "rrset",
					}[k]
					if len(suffix) > 0 {
						path = fmt.Sprintf("/2013-04-01/%s/%s", strings.TrimPrefix(id, "/"), suffix)
					}
					delete(params, "Id")
				}
			}
		}
	}

	op := &request.Operation{
		Name:       apiName,
		HTTPMethod: method,
		HTTPPath:   path,
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
