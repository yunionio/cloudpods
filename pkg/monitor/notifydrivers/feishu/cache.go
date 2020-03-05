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

package feishu

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

type IExpirable interface {
	CreatedAt() int64
	ExpiresIn() int64
}

type ICache interface {
	Set(data IExpirable) error
	Get(data IExpirable) error
}

type FileCache struct {
	Path string
}

func NewFileCache(path string) *FileCache {
	return &FileCache{
		Path: path,
	}
}

func (c *FileCache) Set(data IExpirable) error {
	bytes, err := json.Marshal(data)
	if err == nil {
		ioutil.WriteFile(c.Path, bytes, 0644)
	}
	return err
}

func (c *FileCache) Get(data IExpirable) error {
	bytes, err := ioutil.ReadFile(c.Path)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, data)
	if err != nil {
		return err
	}
	created := data.CreatedAt()
	expires := data.ExpiresIn()
	// The operator '-120' can give us a head start on the expiration date
	if time.Now().Unix() > created+expires-120 {
		err = fmt.Errorf("Data is already expired")
	}
	return err
}
