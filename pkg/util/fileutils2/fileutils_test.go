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
	"io/ioutil"
	"os"
	"testing"
)

// TODO: rewrite this test
/*
func TestIsBlockDeviceUsed(t *testing.T) {
	type args struct {
		dev string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "nbd1",
			args: args{"/dev/nbd1"},
			want: false,
		},
		{
			name: "sda",
			args: args{"/dev/sda"},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBlockDeviceUsed(tt.args.dev); got != tt.want {
				t.Errorf("IsBlockDeviceUsed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetDevId(t *testing.T) {
	type args struct {
		spath string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetDevId(tt.args.spath); got != tt.want {
				t.Errorf("GetDevId() = %v, want %v", got, tt.want)
			}
		})
	}
}*/

func TestGetAllBlkdevsIoScheduler(t *testing.T) {
	scheds, _ := GetAllBlkdevsIoSchedulers()
	t.Logf("scheduler: %#v", scheds)
}

func TestFilePutContents(t *testing.T) {
	cases := []struct {
		isAppend bool
		content  string
		want     string
	}{
		{
			isAppend: false,
			content:  "123",
			want:     "123",
		},
		{
			isAppend: false,
			content:  "abc",
			want:     "abc",
		},
		{
			isAppend: true,
			content:  "123",
			want:     "abc123",
		},
		{
			isAppend: true,
			content:  "abc",
			want:     "abc123abc",
		},
		{
			isAppend: false,
			content:  "123",
			want:     "123",
		},
	}
	file, err := ioutil.TempFile("/tmp", "test*.tmp")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(file.Name())

	t.Log(file.Name())
	for _, c := range cases {
		err := FilePutContents(file.Name(), c.content, c.isAppend)
		if err != nil {
			t.Errorf("FilePutContents fail %s", err)
		} else {
			cont, err := FileGetContents(file.Name())
			if err != nil {
				t.Errorf("FileGetContents %s", err)
			} else if cont != c.want {
				t.Errorf("expect %s got %s", c.want, cont)
			}
		}
	}
}
