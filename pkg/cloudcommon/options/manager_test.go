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

package options

import (
	"reflect"
	"testing"

	"yunion.io/x/jsonutils"
)

func TestCopyOptions(t *testing.T) {
	opts := HostCommonOptions{}
	opts.Port = 8885
	opts.Address = "192.168.22.17"
	opts.LogLevel = "debug"
	opts.AdminUser = "host"
	opts.AdminDomain = "Default"
	opts.AdminPassword = "password"
	opts.AdminProject = "system"
	opts.AdminProjectDomain = "Default"

	newOpts := HostCommonOptions{}
	copyOptions(&newOpts, &opts)
	if !reflect.DeepEqual(opts, newOpts) {
		t.Errorf("opts %s != newOpts %s", jsonutils.Marshal(opts).String(), jsonutils.Marshal(newOpts).String())
	}
}
