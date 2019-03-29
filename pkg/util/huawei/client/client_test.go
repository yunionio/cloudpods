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

package client

import (
	"fmt"
	"testing"
)

func TestClient(t *testing.T) {
	client, err := NewClientWithAccessKey("cn-north-1", "41f6bfe48d7f4455b7754f7c1b11ae34", "XXXXXXXXXXX", "XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX")
	ret, err := client.Projects.List(nil)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(ret)

	r, err := client.Projects.Get("41f6bfe48d7f4455b7754f7c1b11ae34", nil)
	if err != nil {
		fmt.Println(err.Error())
	}

	fmt.Println(r)
}
