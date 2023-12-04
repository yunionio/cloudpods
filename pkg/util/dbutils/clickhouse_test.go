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

package dbutils

import (
	"testing"
)

func TestClickhouseSqlStrV1ToV2(t *testing.T) {
	for _, c := range []struct {
		in   string
		want string
	}{
		{
			in:   "tcp://192.168.222.4:9000?database=yunionmeter&read_timeout=10&write_timeout=20",
			want: "clickhouse://192.168.222.4:9000/yunionmeter?dial_timeout=200ms&max_execution_time=60",
		},
		{
			in:   "tcp://192.168.222.4:9000?username=admin&database=yunionmeter&read_timeout=10&write_timeout=20",
			want: "clickhouse://admin@192.168.222.4:9000/yunionmeter?dial_timeout=200ms&max_execution_time=60",
		},
		{
			in:   "tcp://192.168.222.4:9000?username=admin&password=pass&database=yunionmeter&read_timeout=10&write_timeout=20",
			want: "clickhouse://admin:pass@192.168.222.4:9000/yunionmeter?dial_timeout=200ms&max_execution_time=60",
		},
	} {
		got, err := ClickhouseSqlStrV1ToV2(c.in)
		if err != nil {
			t.Errorf("%s", err)
		} else if got != c.want {
			t.Errorf("got %s want %s", got, c.want)
		}
	}
}

func TestClickhouseSqlStrV2ToV1(t *testing.T) {
	for _, c := range []struct {
		in   string
		want string
	}{
		{
			in:   "clickhouse://admin:pass@192.168.222.4:9000/yunionmeter",
			want: "tcp://192.168.222.4:9000?database=yunionmeter&password=pass&username=admin&read_timeout=10&write_timeout=20",
		},
		{
			in:   "clickhouse://192.168.222.4:9000/yunionmeter?dial_timeout=200ms&max_execution_time=60",
			want: "tcp://192.168.222.4:9000?database=yunionmeter&read_timeout=10&write_timeout=20",
		},
		{
			in:   "clickhouse://admin@192.168.222.4:9000/yunionmeter?dial_timeout=200ms&max_execution_time=60",
			want: "tcp://192.168.222.4:9000?database=yunionmeter&username=admin&read_timeout=10&write_timeout=20",
		},
		{
			in:   "clickhouse://admin:pass@192.168.222.4:9000/yunionmeter?dial_timeout=200ms&max_execution_time=60",
			want: "tcp://192.168.222.4:9000?database=yunionmeter&password=pass&username=admin&read_timeout=10&write_timeout=20",
		},
	} {
		got, err := ClickhouseSqlStrV2ToV1(c.in)
		if err != nil {
			t.Errorf("%s", err)
		} else if got != c.want {
			t.Errorf("got %s want %s", got, c.want)
		}
	}
}

func TestValidateClickhouseSqlstrV1(t *testing.T) {
	for _, c := range []struct {
		in    string
		valid bool
	}{
		{
			in:    "",
			valid: false,
		},
		{
			in:    "tcp://192.168.222.4:9000?read_timeout=10&write_timeout=20",
			valid: false,
		},
		{
			in:    "tcp://192.168.222.4:9000?database=yunionmeter&read_timeout=10&write_timeout=20",
			valid: true,
		},
		{
			in:    "clickhouse://username:password@host1:9000,host2:9000?dial_timeout=200ms&max_execution_time=60",
			valid: false,
		},
		{
			in:    "clickhouse://username:password@host1:9000,host2:9000/database?dial_timeout=200ms&max_execution_time=60",
			valid: false,
		},
	} {
		err := ValidateClickhouseV1Str(c.in)
		if err != nil && c.valid {
			t.Errorf("%s", err)
		}
	}
}

func TestValidateClickhouseSqlstrV2(t *testing.T) {
	for _, c := range []struct {
		in    string
		valid bool
	}{
		{
			in:    "",
			valid: false,
		},
		{
			in:    "tcp://192.168.222.4:9000?database=yunionmeter&read_timeout=10&write_timeout=20",
			valid: false,
		},
		{
			in:    "clickhouse://username:password@host1:9000,host2:9000?dial_timeout=200ms&max_execution_time=60",
			valid: false,
		},
		{
			in:    "clickhouse://username:password@host1:9000,host2:9000/database?dial_timeout=200ms&max_execution_time=60",
			valid: true,
		},
	} {
		err := ValidateClickhouseV2Str(c.in)
		if err != nil && c.valid {
			t.Errorf("%s", err)
		}
	}
}
