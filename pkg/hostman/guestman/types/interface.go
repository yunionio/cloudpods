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

package types

import (
	"yunion.io/x/jsonutils"
)

// this is mainly to avoid direct improt of
// yunion.io/x/onecloud/pkg/hostman/guestman

type IHealthCheckReactor interface {
	ShutdownServers()
}

var HealthCheckReactor IHealthCheckReactor

type IGuestDescGetter interface {
	GetGuestNicDesc(mac, ip, port, bridge string, isCandidate bool) (jsonutils.JSONObject, jsonutils.JSONObject)
}

var GuestDescGetter IGuestDescGetter
