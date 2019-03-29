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

package mcclient

import (
	"github.com/golang-plus/uuid"

	"yunion.io/x/log"
)

type TokenManager interface {
	Save(token TokenCredential) string
	Get(tid string) TokenCredential
	Remove(tid string)
}

type mapTokenManager struct {
	table map[string]TokenCredential
}

func (this *mapTokenManager) Save(token TokenCredential) string {
	key, e := uuid.NewV4()
	if e != nil {
		log.Fatalf("uuid.NewV4 returns error!")
	}
	kkey := key.String()
	this.table[kkey] = token
	// log.Println("###### Save tid", kkey)
	return kkey
}

func (this *mapTokenManager) Get(tid string) TokenCredential {
	// log.Println("###### Get tid", tid)
	return this.table[tid]
}

func (this *mapTokenManager) Remove(tid string) {
	// log.Println("###### Remove tid", tid)
	delete(this.table, tid)
}

func NewMapTokenManager() TokenManager {
	return &mapTokenManager{table: make(map[string]TokenCredential)}
}
