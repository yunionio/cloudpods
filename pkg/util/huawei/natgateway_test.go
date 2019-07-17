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

package huawei

import (
	"testing"
	"yunion.io/x/log"
)

const (
	PROJECT_ID = "41f6bfe48d7f4455b7754f7c1b11ae34"
	ACCESS_KEY = "BLFG27EGJARKKG9HRKGT"
	SECRET     = "9LzCQcE9JogwQM7t42JaFcFfmPQCuUBdjmXSDibw"
)

var region *SRegion

func TestMain(m *testing.M) {
	huaweiClient, err := NewHuaweiClient("001", "huaweiZyTest", "", ACCESS_KEY, SECRET, PROJECT_ID, true)
	if err != nil {
		log.Fatalln(err)
	}
	regionTmp := huaweiClient.iregions[0]
	region = regionTmp.(*SRegion)
	m.Run()
}

func TestSRegion_GetDNatTable(t *testing.T) {

}

func TestSregion_GetSNatTable(t *testing.T) {

}
