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
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func TestSparseFileWriter(t *testing.T) {
	writeTmp := func(t *testing.T, want []byte, partSizes []int) {
		f, err := ioutil.TempFile("", "sparse-test-")
		if err != nil {
			t.Fatalf("tmpfile: %v", err)
		}
		defer f.Close() // close twice
		defer os.Remove(f.Name())

		func() {
			sf := NewSparseFileWriter(f)
			defer sf.Close()

			wrote := 0
			for i, partSize := range partSizes {
				n, err := sf.Write(want[wrote : wrote+partSize])
				if err != nil {
					t.Fatalf("wrote part %d (%d bytes), written %d, : %v", i, partSize, n, err)
				}
				wrote += partSize
			}
		}()

		got, err := ioutil.ReadFile(f.Name())
		if err != nil {
			t.Fatalf("read tmpfile: %v", err)
		}
		if !bytes.Equal(got, want) {
			// hexdump
			t.Errorf("got != want")
		}
		{
			stat := &unix.Stat_t{}
			err := unix.Lstat(f.Name(), stat)
			if err != nil {
				t.Errorf("stat(%s): %v", f.Name(), err)
			}
			t.Logf("data %d, file size: %d, 512*blocks: %d", len(want), stat.Size, 512*stat.Blocks)
		}
	}
	t.Run("1 nul write", func(t *testing.T) {
		want := make([]byte, 1024*8-1)
		writeTmp(t, want, []int{len(want)})
	})
	t.Run("2 nul write", func(t *testing.T) {
		want := make([]byte, 1024*8-1)
		writeTmp(t, want, []int{4096, len(want) - 4096})
	})
	t.Run("1 almost nul write", func(t *testing.T) {
		want := make([]byte, 4095)
		want[1023] = 1
		writeTmp(t, want, []int{len(want)})
	})
	t.Run("nul-nonnul-nul", func(t *testing.T) {
		want := make([]byte, 4095)
		for i := 1023; i < 2024; i++ {
			want[i] = byte(i)
		}
		writeTmp(t, want, []int{1023, 2024 - 1023, 4095 - 2024})
	})
	t.Run("nonnul-nul-nonnul", func(t *testing.T) {
		want := make([]byte, 4095)
		for i := 0; i < 2024; i++ {
			want[i] = byte(i) | 1
		}
		for i := 3011; i < 4095; i++ {
			want[i] = byte(i) | 1
		}
		writeTmp(t, want, []int{2024, 3011 - 2024, 4095 - 3011})
	})
}
