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

import (
	"io"
	"strings"
	"testing"
)

func TestNewReadSeeker(t *testing.T) {
	testStr := "This is a test reader string"
	seeker, err := NewReadSeeker(strings.NewReader(testStr), int64(len(testStr)))
	if err != nil {
		t.Fatalf("NewReadSeeker error %s", err)
	}
	defer seeker.Close()
	buf1 := make([]byte, 1024)
	n, err := seeker.Read(buf1)
	if n != len(testStr) {
		t.Fatalf("read buf1 error %s", err)
	}
	buf2 := make([]byte, 1024)
	n, err = seeker.Read(buf2)
	if n != 0 {
		t.Fatalf("read buf2 should fail")
	}
	seeker.Seek(0, io.SeekStart)
	n, err = seeker.Read(buf2)
	if n != len(testStr) {
		t.Fatalf("read buf2 error %s", err)
	}
}
