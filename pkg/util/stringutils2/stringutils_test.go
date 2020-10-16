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

package stringutils2

import (
	"reflect"
	"testing"

	"yunion.io/x/pkg/util/osprofile"
)

func TestEscapeString(t *testing.T) {
	type args struct {
		str   string
		pairs [][]string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "normalInput",
			args: args{
				str:   "abcd\n\"Te\\rst\"ddd\"$Test\"aaa\n$TTT",
				pairs: nil,
			},
			want: `abcd\n\"Te\\rst\"ddd\"\$Test\"aaa\n\$TTT`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EscapeString(tt.args.str, tt.args.pairs); got != tt.want {
				t.Errorf("EscapeString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSplitByQuotation(t *testing.T) {
	type args struct {
		line string
	}
	tests := []struct {
		name    string
		args    args
		want    []string
		wantErr bool
	}{
		{
			name:    "normalInput",
			args:    args{`"abc" addf "sada"`},
			want:    []string{"abc", " addf ", "sada"},
			wantErr: false,
		},
		{
			name:    "errorInput",
			args:    args{`"abc" "addf "sada"`},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitByQuotation(tt.args.line)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitByQuotation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitByQuotation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEscapeEchoString(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "normalInput",
			args:    args{"abcd\n\"Te\\rst\"ddd\"$Test\"aaa\n$TTT"},
			want:    `abcd\n\"Te\\\\rst\"ddd\"\$Test\"aaa\n\$TTT`,
			wantErr: false,
		},
		{
			name:    "echoInput",
			args:    args{"SUBSYSTEM==\"usb\", ATTRS{idVendor}==\"1d6b\", ATTRS{idProduct}==\"0001\", RUN+=\"/bin/sh -c 'echo enabled > /sys$env{DEVPATH}/../power/wakeup'\""},
			want:    `SUBSYSTEM==\"usb\", ATTRS{idVendor}==\"1d6b\", ATTRS{idProduct}==\"0001\", RUN+=\"/bin/sh -c 'echo enabled > /sys\$env{DEVPATH}/../power/wakeup'\"`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := EscapeEchoString(tt.args.str)
			if (err != nil) != tt.wantErr {
				t.Errorf("EscapeEchoString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("EscapeEchoString() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateHostName(t *testing.T) {
	type input struct {
		name     string
		osType   string
		hostName string
	}
	for _, pair := range []input{
		input{
			name:     "--test-host-name.......",
			hostName: "test-host-name",
		},
		input{
			name:     "--test-host-1234567890-name.......",
			osType:   osprofile.OS_TYPE_WINDOWS,
			hostName: "test-host-12345",
		},
		input{
			name:     "--test-host-1234-67890-name.......",
			osType:   osprofile.OS_TYPE_WINDOWS,
			hostName: "test-host-1234",
		},
		input{
			name:     "1234567890123456",
			osType:   osprofile.OS_TYPE_WINDOWS,
			hostName: "host-1234567890",
		},
		input{
			name:     "001234567890123456",
			osType:   osprofile.OS_TYPE_WINDOWS,
			hostName: "host-0012345678",
		},

		input{
			name:     "",
			hostName: "hostname-for",
		},
	} {
		hostName := GenerateHostName(pair.name, pair.osType)
		if hostName != pair.hostName {
			t.Fatalf("%s hostName should be %s, current is %s", pair.name, pair.hostName, hostName)
		}
	}
}
