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

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"yunion.io/x/log"
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
