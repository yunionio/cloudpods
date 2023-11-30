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
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/client/metadata"
	"github.com/aws/aws-sdk-go/aws/corehandlers"
	"github.com/aws/aws-sdk-go/aws/request"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

// eks only
func (self *SAwsClient) invoke(regionId, serviceName, serviceId, apiVersion string, apiName, path string, params map[string]interface{}, retval interface{}, assumeRole bool) error {
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
	if serviceName == PRICING_SERVICE_NAME {
		metadata.TargetPrefix = "AWSPriceListService"
		metadata.JSONVersion = "1.1"
	}
	if serviceName == ECS_SERVICE_NAME {
		metadata.TargetPrefix = "AmazonEC2ContainerServiceV20141113"
		metadata.JSONVersion = "1.1"
	}
	if serviceName == KINESIS_SERVICE_NAME {
		metadata.TargetPrefix = "Kinesis_20131202"
		metadata.JSONVersion = "1.1"
	}
	if serviceName == DYNAMODB_SERVICE_NAME {
		metadata.TargetPrefix = "DynamoDB_20120810"
		metadata.JSONVersion = "1.0"
	}

	if self.debug {
		logLevel := aws.LogLevelType(uint(aws.LogDebugWithRequestErrors) + uint(aws.LogDebugWithHTTPBody))
		c.Config.LogLevel = &logLevel
	}

	client := client.New(*c.Config, metadata, c.Handlers)
	client.Handlers.Sign.PushBackNamed(v4.SignRequestHandler)
	client.Handlers.Build.PushBackNamed(JsonBuildHandler)
	client.Handlers.Unmarshal.PushBackNamed(UnmarshalJsonHandler)
	//client.Handlers.UnmarshalMeta.PushBackNamed(restjson.UnmarshalMetaHandler)
	client.Handlers.UnmarshalError.PushBackNamed(UnmarshalJsonErrorHandler)
	client.Handlers.Validate.Remove(corehandlers.ValidateEndpointHandler)
	return jsonInvoke(client, apiName, path, params, retval, true)
}

func jsonInvoke(cli *client.Client, apiName, path string, params map[string]interface{}, retval interface{}, debug bool) error {
	method := "POST"
	for _, key := range []string{"List", "Describe", "Get"} {
		if strings.HasPrefix(apiName, key) {
			method = "GET"
			break
		}
	}
	if strings.HasPrefix(apiName, "Delete") {
		method = "DELETE"
	}
	if method == "GET" || method == "DELETE" {
		for k, v := range params {
			if strings.Contains(path, "{"+k+"}") {
				path = strings.Replace(path, "{"+k+"}", v.(string), 1)
				delete(params, k)
			}
		}
	}
	if utils.IsInStringArray(cli.ServiceName, []string{
		ECS_SERVICE_NAME,
		KINESIS_SERVICE_NAME,
		DYNAMODB_SERVICE_NAME,
	}) {
		method = "POST"
	}

	op := &request.Operation{
		Name:       apiName,
		HTTPMethod: method,
		HTTPPath:   path,
		Paginator: &request.Paginator{
			InputTokens:     []string{"nextToken"},
			OutputTokens:    []string{"nextToken"},
			LimitToken:      "maxResults",
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

var JsonBuildHandler = request.NamedHandler{
	Name: "yunion.json.Build",
	Fn:   JsonBuild,
}

func JsonBuild(r *request.Request) {
	if r.Config.LogLevel != nil && r.Config.LogLevel.AtLeast(aws.LogDebugWithHTTPBody) {
		log.Debugf("body: %s", jsonutils.Marshal(r.Params).PrettyString())
	}

	if r.Params != nil && r.HTTPRequest.Method == "POST" {
		r.SetBufferBody([]byte(jsonutils.Marshal(r.Params).String()))
	}

	if len(r.ClientInfo.APIVersion) > 0 {
		jsonVersion := r.ClientInfo.JSONVersion
		r.HTTPRequest.Header.Set("Content-Type", "application/x-amz-json-"+jsonVersion)
	}

	if v := r.HTTPRequest.Header.Get("Content-Type"); len(v) == 0 {
		r.HTTPRequest.Header.Set("Content-Type", "application/json")
	}

	if r.ClientInfo.TargetPrefix != "" {
		target := r.ClientInfo.TargetPrefix + "." + r.Operation.Name
		r.HTTPRequest.Header.Add("X-Amz-Target", target)
	}

}

var UnmarshalJsonHandler = request.NamedHandler{Name: "yunion.query.UnmarshalJson", Fn: UnmarshalJson}

func UnmarshalJson(r *request.Request) {
	defer r.HTTPResponse.Body.Close()
	if r.DataFilled() {
		body, err := ioutil.ReadAll(r.HTTPResponse.Body)
		if err != nil {
			r.Error = awserr.NewRequestFailure(
				awserr.New("ioutil.ReadAll", "read response body", err),
				r.HTTPResponse.StatusCode,
				r.RequestID,
			)
			return
		}

		if r.Config.LogLevel != nil && r.Config.LogLevel.AtLeast(aws.LogDebugWithHTTPBody) {
			log.Debugf("response: \n%s", string(body))
		}

		obj, err := jsonutils.Parse(body)
		if err != nil {
			r.Error = awserr.NewRequestFailure(
				awserr.New("DecodeElement", "failed decoding Query response", err),
				r.HTTPResponse.StatusCode,
				r.RequestID,
			)
			return
		}

		err = obj.Unmarshal(r.Data)
		if err != nil {
			r.Error = awserr.NewRequestFailure(
				awserr.New("DecodeElement", "failed decoding Query response", err),
				r.HTTPResponse.StatusCode,
				r.RequestID,
			)
			return
		}
	}
}

var UnmarshalJsonErrorHandler = request.NamedHandler{
	Name: "awssdk.ec2query.UnmarshalJsonError",
	Fn:   UnmarshalJsonError,
}

type sAwsInvokeError struct {
	Message string
	Type    string `json:"__type"`
}

func (self sAwsInvokeError) Error() string {
	return jsonutils.Marshal(self).String()
}

func UnmarshalJsonError(r *request.Request) {
	defer r.HTTPResponse.Body.Close()

	result, err := ioutil.ReadAll(r.HTTPResponse.Body)
	if err != nil {
		r.Error = errors.Wrapf(err, "ioutil.ReadAll")
		return
	}

	if r.HTTPResponse.StatusCode == 404 {
		r.Error = errors.Wrapf(cloudprovider.ErrNotFound, string(result))
		return
	}

	obj, err := jsonutils.Parse(result)
	if err != nil {
		r.Error = errors.Wrapf(err, "jsonutils.Parse")
		return
	}

	if r.Config.LogLevel != nil && r.Config.LogLevel.AtLeast(aws.LogDebugWithHTTPBody) {
		log.Debugf("response error: %s", obj.PrettyString())
	}

	respErr := &sAwsInvokeError{}
	err = obj.Unmarshal(respErr)
	if err != nil {
		r.Error = errors.Wrapf(err, obj.String())
		return
	}

	r.Error = respErr
	return
}
