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

package jsonutils

import (
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/pkg/utils"
)

func (this *JSONDict) parseQueryString(str string) error {
	m, err := url.ParseQuery(str)
	if err != nil {
		return errors.Wrap(err, "url.ParseQuery")
	}
	for k, v := range m {
		keys := strings.Split(k, ".")
		if len(v) == 1 {
			this.Add(NewString(v[0]), keys...)
		} else if len(v) > 1 {
			arr := NewArray()
			for _, val := range v {
				arr.Add(NewString(val))
			}
			this.Add(arr, keys...)
		}
	}
	return nil
}

func ParseQueryString(str string) (JSONObject, error) {
	dict := NewDict()
	err := dict.parseQueryString(str)
	if err != nil {
		return nil, errors.Wrap(err, "dict.parseQueryString")
	}
	return dict, nil
}

func simpleQueryString(key, val string) string {
	if len(key) > 0 && len(val) > 0 {
		return url.QueryEscape(key) + "=" + url.QueryEscape(val)
	} else if len(val) > 0 {
		return url.QueryEscape(val)
	} else if len(key) > 0 {
		return url.QueryEscape(key)
	} else {
		return ""
	}
}

func (this *JSONValue) _queryString(key string) string {
	return simpleQueryString(key, "")
}

func (this *JSONString) _queryString(key string) string {
	return simpleQueryString(key, this.data)
}

func (this *JSONInt) _queryString(key string) string {
	return simpleQueryString(key, this.String())
}

func (this *JSONFloat) _queryString(key string) string {
	return simpleQueryString(key, this.String())
}

func (this *JSONBool) _queryString(key string) string {
	return simpleQueryString(key, this.String())
}

func (this *JSONArray) _queryString(key string) string {
	rets := make([]string, 0)
	for _, val := range this.data {
		/* k := fmt.Sprintf("%d", i)
		   if len(key) > 0 {
		       k = key + "." + k
		   } */
		rets = append(rets, val._queryString(key))
	}
	return strings.Join(rets, "&")
}

func (this *JSONDict) _queryString(key string) string {
	rets := make([]string, 0)
	for _, k := range this.SortedKeys() {
		v := this.data[k]
		if len(key) > 0 {
			k = key + "." + k
		}
		rets = append(rets, v._queryString(k))
	}
	return strings.Join(rets, "&")
}

func (this *JSONValue) QueryString() string {
	return this._queryString("")
}

func (this *JSONDict) QueryString() string {
	return this._queryString("")
}

func QueryBoolean(query JSONObject, key string, defVal bool) bool {
	if query == nil {
		return defVal
	}
	jsonVal, _ := query.Get(key)
	if jsonVal != nil {
		str, _ := jsonVal.GetString()
		return utils.ToBool(str)
	} else {
		return defVal
	}
}
