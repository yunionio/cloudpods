package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"yunion.io/x/jsonutils"

	agentutils "yunion.io/x/onecloud/pkg/lbagent/utils"
	"yunion.io/x/onecloud/pkg/mcclient"
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
		ModelSets:             NewModelSets(),
		ModelSetsMaxUpdatedAt: NewModelSetsMaxUpdatedAt(),
	}
}

func (b *LoadbalancerCorpus) SyncModelSets(s *mcclient.ClientSession, batchSize int) (*ModelSetsUpdateResult, error) {
	mss := b.ModelSetList()
	mssNews := NewModelSets()
	for i, msNew := range mssNews.ModelSetList() {
		minUpdatedAt := ModelSetMaxUpdatedAt(mss[i])
		err := GetModels(&GetModelsOptions{
			ClientSession: s,
			ModelManager:  msNew.ModelManager(),
			MinUpdatedAt:  minUpdatedAt,
			ModelSet:      msNew,
			BatchListSize: batchSize,
		})
		if err != nil {
			return nil, err
		}
	}
	r := b.ModelSets.ApplyUpdates(mssNews)
	b.ModelSetsMaxUpdatedAt = r.ModelSetsMaxUpdatedAt
	return r, nil
}

func (b *LoadbalancerCorpus) MaxSeenUpdatedAtParams() *jsonutils.JSONDict {
	mssmua := b.ModelSetsMaxUpdatedAt
	return jsonutils.Marshal(mssmua).(*jsonutils.JSONDict)
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
