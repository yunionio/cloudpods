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

package esxi

import "testing"

func TestInitValue(t *testing.T) {
	type testStruct struct {
		member1 int
		member2 string
		Member3 int
		Member4 string
	}

	dst := testStruct{}

	dst.member1 = 1
	dst.member2 = "2"
	dst.Member3 = 3
	dst.Member4 = "4"

	pDst := &dst
	t.Logf("%p %#v", pDst, pDst)

	*pDst = testStruct{}

	t.Logf("%p %#v", pDst, pDst)

	new := testStruct{}
	if dst != new {
		t.Errorf("dst != new")
	}
}
