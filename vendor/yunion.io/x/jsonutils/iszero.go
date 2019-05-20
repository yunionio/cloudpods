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

func (this *JSONValue) IsZero() bool {
	return true
}

func (this *JSONBool) IsZero() bool {
	return this.data == false
}

func (this *JSONInt) IsZero() bool {
	return this.data == 0
}

func (this *JSONFloat) IsZero() bool {
	return this.data == 0.0
}

func (this *JSONString) IsZero() bool {
	return len(this.data) == 0
}

func (this *JSONDict) IsZero() bool {
	return this.data == nil || len(this.data) == 0
}

func (this *JSONArray) IsZero() bool {
	return this.data == nil || len(this.data) == 0
}
