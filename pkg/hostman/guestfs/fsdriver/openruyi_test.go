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

package fsdriver

import (
	"reflect"
	"testing"
)

func TestOpenRuyiRootSignatures(t *testing.T) {
	driver := NewOpenRuyiRootFs(nil)
	want := []string{
		"/etc/os-release",
		"/usr/lib/rpm/openruyi",
		"/bin",
		"/etc",
		"/boot",
		"/lib",
		"/usr",
	}
	if got := driver.RootSignatures(); !reflect.DeepEqual(got, want) {
		t.Fatalf("RootSignatures() = %#v, want %#v", got, want)
	}
}

func TestParseOpenRuyiVersion(t *testing.T) {
	tests := []struct {
		name      string
		osRelease string
		want      string
	}{
		{name: "double quoted", osRelease: "NAME=\"openRuyi\"\nVERSION_ID=\"Creek\"\n", want: "Creek"},
		{name: "single quoted", osRelease: "VERSION_ID='River'\n", want: "River"},
		{name: "unquoted", osRelease: "VERSION_ID=Lake\n", want: "Lake"},
		{name: "whitespace and CRLF", osRelease: "NAME=openRuyi\r\n  VERSION_ID = ignored\r\n VERSION_ID=\"Creek\"  \r\n", want: "Creek"},
		{name: "similarly named key", osRelease: "BUILD_VERSION_ID=Creek\n", want: ""},
		{name: "empty value", osRelease: "VERSION_ID=\"\"\n", want: ""},
		{name: "missing", osRelease: "NAME=\"openRuyi\"\n", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseOpenRuyiVersion(tt.osRelease); got != tt.want {
				t.Fatalf("parseOpenRuyiVersion() = %q, want %q", got, tt.want)
			}
		})
	}
}
