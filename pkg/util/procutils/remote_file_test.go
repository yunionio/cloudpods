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

package procutils

import (
	"testing"
)

func TestFilePutContents(t *testing.T) {
	content := `this is a test string
including text test
including text test
including text test
including text test
including text test
including text test
including text test
including text test
including text test`
	SetRemoteTempDir("/tmp/files")
	err := FilePutContents("/tmp/test", content)
	if err != nil {
		t.Errorf("FilePutContents failed: %v", err)
	} else {
		content2, err := FileGetContents("/tmp/test")
		if err != nil {
			t.Errorf("FileGetContents failed: %v", err)
		} else if string(content2) != content {
			t.Errorf("FileGetContents expect %s got %s", content, string(content2))
		}
	}
}
