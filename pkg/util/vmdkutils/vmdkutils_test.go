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

package vmdkutils

import "testing"

const (
	VMDKContent = `    version=1
    encoding="UTF-8"
    CID=ab798ecb
    parentCID=ffffffff
    isNativeSnapshot="no"
    createType="vmfs"

    # Extent description
    RW 62914560 VMFS "c8336ec2-3885-4205-a2f1-095e48228a56-flat.vmdk"

    # The Disk Data Base
    #DDB

    ddb.adapterType = "lsilogic"
    ddb.deletable = "false"
    ddb.geometry.cylinders = "62415"
    ddb.geometry.heads = "16"
    ddb.geometry.sectors = "63"
    ddb.longContentID = "4d1762d22407e6b7e3e63557ab798ecb"
    ddb.thinProvisioned = "1"
    ddb.uuid = "60 00 C2 97 32 7d ac 2b-ac 97 a4 95 5b a6 0b c0"
    ddb.virtualHWVersion = "13"
`
)

func TestParseStream(t *testing.T) {
	info, err := Parse(VMDKContent)
	if err != nil {
		t.Errorf("parse error %s", err)
	}
	t.Logf("%#v", info)

	_, err = Parse("")
	if err != nil {
		t.Logf("parse error %s", err)
	} else {
		t.Errorf("should parse error")
	}
}
