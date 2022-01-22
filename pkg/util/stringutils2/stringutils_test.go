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

func TestGetCharTypeCount(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{
			in:   "",
			want: 0,
		},
		{
			in:   "123",
			want: 1,
		},
		{
			in:   "abc",
			want: 1,
		},
		{
			in:   "abcAbc",
			want: 2,
		},
		{
			in:   "123dbA",
			want: 3,
		},
		{
			in:   "123@Acv",
			want: 4,
		},
		{
			in:   "中文",
			want: 1,
		},
	}
	for _, c := range cases {
		got := GetCharTypeCount(c.in)
		if got != c.want {
			t.Errorf("GetCharTypeCount %s want %d got %d", c.in, c.want, got)
		}
	}
}

func TestRoleName(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		random bool
		length int
	}{
		{
			in:     "小琪",
			random: true,
			length: 17,
		},
		{
			in:   "123^567",
			want: "123567",
		},
	}
	for _, c := range cases {
		got := GenerateRoleName(c.in)
		if c.random {
			if len(got) != c.length {
				t.Errorf("GenerateRoleName %s random want %d length got %d(%s)", c.in, c.length, len(got), got)
			}
		} else if got != c.want {
			t.Errorf("GenerateRoleName %s want %s got %s", c.in, c.want, got)
		}
	}
}

func TestFilterEmpty(t *testing.T) {
	cases := []struct {
		input []string
		want  []string
	}{
		{
			input: []string{
				"",
			},
			want: []string{},
		},
	}
	for _, c := range cases {
		got := FilterEmpty(c.input)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("want: %#v got: %#v", c.want, got)
		}
	}
}
