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

package qemutils

import (
	"testing"

	"yunion.io/x/log"
)

func TestGetQemu(t *testing.T) {
	log.Infof("default: %s", GetQemu(""))
	log.Infof("2.9.1: %s", GetQemu("2.9.1"))
	log.Infof("2.12.1: %s", GetQemu("2.12.1"))
	log.Infof("2.12.2: %s", GetQemu("2.12.2"))
	log.Infof(GetQemuImg())
	log.Infof(GetQemuNbd())
}
