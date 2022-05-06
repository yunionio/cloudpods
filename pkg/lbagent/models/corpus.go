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

package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"yunion.io/x/jsonutils"

	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
)

const (
	CORPUS_VERSION = "v1"
)

type LoadbalancerCorpus struct {
	CorpusVersion string
	*ModelSets
	ModelSetsMaxUpdatedAt *ModelSetsMaxUpdatedAt
}

func NewEmptyLoadbalancerCorpus() *LoadbalancerCorpus {
	return &LoadbalancerCorpus{
		CorpusVersion:         CORPUS_VERSION,
		ModelSets:             NewModelSets(),
		ModelSetsMaxUpdatedAt: NewModelSetsMaxUpdatedAt(),
	}
}

func (b *LoadbalancerCorpus) MaxSeenUpdatedAtParams() *jsonutils.JSONDict {
	return b.ModelSets.MaxSeenUpdatedAtParams()
}

func (b *LoadbalancerCorpus) SaveDir(dir string) error {
	d, err := json.Marshal(b)
	if err != nil {
		return err
	}
	p := filepath.Join(dir, "corpus")
	err = ioutil.WriteFile(p, d, agentutils.FileModeFileSensitive)
	return err
}

func (b *LoadbalancerCorpus) LoadDir(dir string) error {
	p := filepath.Join(dir, "corpus")
	d, err := ioutil.ReadFile(p)
	if err != nil {
		return err
	}
	err = json.Unmarshal(d, b)
	if err != nil {
		return err
	}
	// version for updating
	if ver := b.CorpusVersion; ver != CORPUS_VERSION {
		b.Reset()
		return fmt.Errorf("%s: corpus version %s != %s", p, ver, CORPUS_VERSION)
	}
	correct := b.join()
	if !correct {
		return fmt.Errorf("%s: corpus data has inconsistencies", p)
	}
	return nil
}

func (b *LoadbalancerCorpus) Reset() {
	bb := NewEmptyLoadbalancerCorpus()
	*b = *bb
}
