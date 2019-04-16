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

package influxdb

// TODO: rewrite this test
/*
import (
	"testing"
)

func TestInfluxdb(t *testing.T) {
	url := "https://192.168.222.171:8086"

	db := NewInfluxdb(url)
	err := db.SetDatabase("telegraf1")
	if err != nil {
		t.Fatalf("GetDatabases: %s", err)
	}
	rp := SRetentionPolicy{
		Name:     "30days",
		Duration: "30d",
		ReplicaN: 1,
		Default:  true,
	}
	err = db.SetRetentionPolicy(rp)
	if err != nil {
		t.Fatalf("SetRetentPolicy: %s", err)
	}
	rps, err := db.GetRetentionPolicies()
	if err != nil {
		t.Fatalf("GetRentionPolicies %s", err)
	}
	t.Logf("%#v", rps)
}*/
