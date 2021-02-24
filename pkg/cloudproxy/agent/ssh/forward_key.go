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

package ssh

import (
	"fmt"
)

const (
	ForwardKeyTypeL = "L"
	ForwardKeyTypeR = "R"
)

type ForwardKey struct {
	EpKey   string
	Type    string
	KeyAddr string
	KeyPort int

	Value interface{}
}

func (fk *ForwardKey) Key() string {
	return fmt.Sprintf("%s/%s/%s/%d", fk.EpKey, fk.Type, fk.KeyAddr, fk.KeyPort)
}

type ForwardKeySet map[string]ForwardKey

func (fks ForwardKeySet) addByPortMap(epKey, typ string, pm portMap) {
	for port, addrMap := range pm {
		for addr := range addrMap {
			fk := ForwardKey{
				EpKey:   epKey,
				Type:    typ,
				KeyAddr: addr,
				KeyPort: port,
			}
			fks.Add(fk)
		}
	}
}

func (fks ForwardKeySet) Contains(fk ForwardKey) bool {
	key := fk.Key()
	if _, ok := fks[key]; ok {
		return true
	}
	return false
}

func (fks ForwardKeySet) Remove(fk ForwardKey) {
	delete(fks, fk.Key())
}

func (fks ForwardKeySet) Add(fk ForwardKey) {
	fks[fk.Key()] = fk
}
