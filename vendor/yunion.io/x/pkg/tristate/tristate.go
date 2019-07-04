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

package tristate

import "reflect"

type TriState string

const (
	True  = TriState("true")
	False = TriState("false")
	None  = TriState("")
)

var (
	TriStateType       = reflect.TypeOf(True)
	TriStateTrueValue  = reflect.ValueOf(True)
	TriStateFalseValue = reflect.ValueOf(False)
	TriStateNoneValue  = reflect.ValueOf(None)
)

func (r TriState) Bool() bool {
	switch r {
	case True:
		return true
	case False:
		return false
	default:
		return false
	}
}

func (r TriState) String() string {
	return string(r)
}

func (r TriState) IsNone() bool {
	return r == None
}

func (r TriState) IsTrue() bool {
	return r == True
}

func (r TriState) IsFalse() bool {
	return r == False
}

func NewFromBool(b bool) TriState {
	if b {
		return True
	} else {
		return False
	}
}
