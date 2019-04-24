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

package fileutils2

import "testing"

func TestGetBlkidType(t *testing.T) {
	cases := []struct {
		In   string
		Want string
	}{
		{
			`/dev/sda2: UUID="87a523a8-b382-4b45-a291-7ae56a13c99a" TYPE="ext4" PARTLABEL="Linux" PARTUUID="d9ac3dd7-da80-4c57-a701-e37956c07687"`,
			"ext4",
		},
		{
			`/opt/isoimage/iso/yunion-20180622.iso: UUID="2018-06-22-23-04-12-00" LABEL="CDROM" TYPE="iso9660" PTTYPE="dos"`,
			"iso9660",
		},
	}
	for _, c := range cases {
		matches := blkidTypeRegexp.FindStringSubmatch(c.In)
		if len(matches) > 1 && matches[1] == c.Want {
			t.Logf("%s", matches)
		} else {
			t.Errorf("fail")
		}
	}
}
