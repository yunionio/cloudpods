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
	"yunion.io/x/pkg/sortedmap"
)

func (self *JSONValue) Interface() interface{} {
	return nil
}

func (self *JSONBool) Interface() interface{} {
	return self.data
}

func (self *JSONInt) Interface() interface{} {
	return self.data
}

func (self *JSONFloat) Interface() interface{} {
	return self.data
}

func (self *JSONString) Interface() interface{} {
	return self.data
}

func (self *JSONArray) Interface() interface{} {
	ret := make([]interface{}, len(self.data))
	for i := 0; i < len(self.data); i += 1 {
		ret[i] = self.data[i].Interface()
	}
	return ret
}

func (self *JSONDict) Interface() interface{} {
	mapping := make(map[string]interface{})

	for iter := sortedmap.NewIterator(self.data); iter.HasMore(); iter.Next() {
		k, v := iter.Get()
		mapping[k] = v.(JSONObject).Interface()
	}

	return mapping
}
