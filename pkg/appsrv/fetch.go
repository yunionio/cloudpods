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

package appsrv

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
)

func Fetch(req *http.Request) ([]byte, error) {
	defer req.Body.Close()
	return ioutil.ReadAll(req.Body)
}

func FetchStruct(req *http.Request, v interface{}) error {
	b, e := Fetch(req)
	if e != nil {
		return e
	}
	if len(b) > 0 {
		obj, err := jsonutils.Parse(b)
		if err != nil {
			return err
		}
		return obj.Unmarshal(v)
	} else {
		return nil
	}
}

func FetchJSON(req *http.Request) (jsonutils.JSONObject, error) {
	b, e := Fetch(req)
	if e != nil {
		return nil, e
	}
	if len(b) > 0 {
		return jsonutils.Parse(b)
	} else {
		return nil, nil
	}
}

func FetchXml(req *http.Request, target interface{}) error {
	b, e := Fetch(req)
	if e != nil {
		return errors.Wrap(e, "Fetch")
	}
	if len(b) > 0 {
		return xml.Unmarshal(b, target)
	} else {
		return nil
	}
}
