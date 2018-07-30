package data_manager

import (
	"fmt"
	"time"

	"github.com/yunionio/onecloud/pkg/scheduler/cache"
	candidatecache "github.com/yunionio/onecloud/pkg/scheduler/cache/candidate"
	dbcache "github.com/yunionio/onecloud/pkg/scheduler/cache/db"
	synccache "github.com/yunionio/onecloud/pkg/scheduler/cache/sync"
	"github.com/yunionio/onecloud/pkg/scheduler/core"
	"github.com/yunionio/pkg/util/ttlpool"
)

type CandidateGetArgs struct {
	ResType    string
	ZoneID     string
	PoolID     string
	IgnorePool bool
}

type DataManager struct {
	DBCacheGroup   cache.CacheGroup
	SyncCacheGroup cache.CacheGroup
	CandidateGroup cache.CacheGroup
}

func NewDataManager(stopCh <-chan struct{}) *DataManager {
	m := new(DataManager)
	m.DBCacheGroup = dbcache.NewCacheManager(stopCh)
	m.SyncCacheGroup = synccache.NewSyncManager(stopCh)
	m.CandidateGroup = candidatecache.NewCandidateManager(
		m.DBCacheGroup, m.SyncCacheGroup, stopCh)

	return m
}

func (m *DataManager) Run() {
	go m.DBCacheGroup.Run()
	go m.SyncCacheGroup.Run()
	go m.CandidateGroup.Run()
}

type CandidateManagerImplProvider interface {
	LoadCandidates() ([]interface{}, error)
	ReloadCandidates(ids []string) ([]interface{}, error)
	ReloadAllCandidates() ([]interface{}, error)
	GetCandidate(id string) (interface{}, error)
}

type HostCandidateManagerImplProvider struct {
	dataManager *DataManager
}

func getCache(dataManager *DataManager, name string) (cache.Cache, error) {
	candidate_cache, err := dataManager.CandidateGroup.Get(name)
	if err != nil {
		return nil, err
	}

	candidate_cache.WaitForReady()
	return candidate_cache, nil
}

func (p *HostCandidateManagerImplProvider) LoadCandidates() ([]interface{}, error) {
	candidate_cache, err := getCache(p.dataManager, candidatecache.HostCandidateCache)
	if err != nil {
		return nil, err
	}

	return candidate_cache.List(), nil
}

func (p *HostCandidateManagerImplProvider) ReloadCandidates(
	ids []string) ([]interface{}, error) {
	candidate_cache, err := getCache(p.dataManager, candidatecache.HostCandidateCache)
	if err != nil {
		return nil, err
	}

	return candidate_cache.Reload(ids)
}

func (p *HostCandidateManagerImplProvider) ReloadAllCandidates() ([]interface{}, error) {
	candidate_cache, err := getCache(p.dataManager, candidatecache.HostCandidateCache)
	if err != nil {
		return nil, err
	}

	return candidate_cache.ReloadAll()
}

func (p *HostCandidateManagerImplProvider) GetCandidate(id string) (interface{}, error) {
	candidate_cache, err := getCache(p.dataManager, candidatecache.HostCandidateCache)
	if err != nil {
		return nil, err
	}

	return candidate_cache.Get(id)
}

type BaremetalCandidateManagerImplProvider struct {
	dataManager *DataManager
}

func (p *BaremetalCandidateManagerImplProvider) LoadCandidates() ([]interface{}, error) {
	candidate_cache, err := getCache(p.dataManager, candidatecache.BaremetalCandidateCache)
	if err != nil {
		return nil, err
	}

	return candidate_cache.List(), nil
}

func (p *BaremetalCandidateManagerImplProvider) ReloadCandidates(
	ids []string) ([]interface{}, error) {
	candidate_cache, err := getCache(p.dataManager, candidatecache.BaremetalCandidateCache)
	if err != nil {
		return nil, err
	}

	return candidate_cache.Reload(ids)
}

func (p *BaremetalCandidateManagerImplProvider) ReloadAllCandidates() ([]interface{}, error) {
	candidate_cache, err := getCache(p.dataManager, candidatecache.BaremetalCandidateCache)
	if err != nil {
		return nil, err
	}

	return candidate_cache.ReloadAll()
}

func (p *BaremetalCandidateManagerImplProvider) GetCandidate(id string) (interface{}, error) {
	candidate_cache, err := getCache(p.dataManager, candidatecache.BaremetalCandidateCache)
	if err != nil {
		return nil, err
	}

	return candidate_cache.Get(id)
}

type CandidateManagerImpl struct {
	provider     CandidateManagerImplProvider
	dataMap      map[string][]interface{}
	stopCh       <-chan struct{}
	lastLoadTime time.Time
}

func NewCandidateManagerImpl(provider CandidateManagerImplProvider, stopCh <-chan struct{},
) *CandidateManagerImpl {
	return &CandidateManagerImpl{
		provider: provider,
		dataMap:  make(map[string][]interface{}),
		stopCh:   stopCh,
	}
}

func (impl *CandidateManagerImpl) GetCandidates() ([]interface{}, error) {
	return impl.provider.LoadCandidates()
}

func (impl *CandidateManagerImpl) GetCandidate(id string) (interface{}, error) {
	return impl.provider.GetCandidate(id)
}

func (impl *CandidateManagerImpl) Reload(ids []string) ([]interface{}, error) {
	return impl.provider.ReloadCandidates(ids)
}

func (impl *CandidateManagerImpl) ReloadAll() ([]interface{}, error) {
	return impl.provider.ReloadAllCandidates()
}

func (impl *CandidateManagerImpl) Run() {
}

type CandidateManager struct {
	stopCh      <-chan struct{}
	dataManager *DataManager
	impls       map[string]*CandidateManagerImpl

	dirtyPool *ttlpool.CountPool
}

func (cm *CandidateManager) DirtyPoolHas(id string) bool {
	ok, _ := cm.dirtyPool.HasByKey(id)
	return ok
}

func (cm *CandidateManager) GetCandidates(args CandidateGetArgs) ([]core.Candidater, error) {
	impl, err := cm.getImpl(args.ResType)
	if err != nil {
		return nil, err
	}

	candidates, err2 := impl.GetCandidates()
	if err2 != nil {
		return nil, err2
	}

	hasZone := len(args.ZoneID) > 0

	result := []core.Candidater{}

	for _, c := range candidates {
		r := c.(core.Candidater)

		if cm.DirtyPoolHas(r.IndexKey()) {
			continue
		}

		if args.IgnorePool {
			result = append(result, r)
		} else if (!hasZone || r.Get("ZoneID") == args.ZoneID) && r.Get("PoolID") == args.PoolID {
			result = append(result, r)
		}
	}

	return result, nil
}

func (cm *CandidateManager) GetCandidatesByIds(resType string, ids []string) ([]core.Candidater, error) {
	impl, err := cm.getImpl(resType)
	if err != nil {
		return nil, err
	}

	candidates := []core.Candidater{}
	for _, id := range ids {
		if cm.DirtyPoolHas(id) {
			continue
		}

		c, err2 := impl.GetCandidate(id)
		if err2 != nil {
			return nil, err2
		}
		candidates = append(candidates, c.(core.Candidater))
	}

	return candidates, nil
}

func (cm *CandidateManager) GetCandidate(id string, resType string) (interface{}, error) {
	impl, err := cm.getImpl(resType)
	if err != nil {
		return nil, err
	}

	if cm.DirtyPoolHas(id) {
		return nil, fmt.Errorf("%s in dirtyPool", id)
	}

	c, err := impl.GetCandidate(id)
	if err != nil {
		return nil, err
	}
	return c.(core.Candidater), nil
}

func (cm *CandidateManager) getImpl(resType string) (*CandidateManagerImpl, error) {
	var (
		impl *CandidateManagerImpl
		ok   bool
	)

	if impl, ok = cm.impls[resType]; !ok {
		return nil, fmt.Errorf("Resource Type \"%v\" not supported", resType)
	}

	return impl, nil
}

func (cm *CandidateManager) AddImpl(name string, impl *CandidateManagerImpl) {
	cm.impls[name] = impl
}

func NewCandidateManager(dataManager *DataManager, stopCh <-chan struct{}) *CandidateManager {

	candidateManager := &CandidateManager{
		stopCh:      stopCh,
		impls:       make(map[string]*CandidateManagerImpl),
		dataManager: dataManager,
		dirtyPool:   ttlpool.NewCountPool(),
	}

	candidateManager.AddImpl("host", NewCandidateManagerImpl(
		&HostCandidateManagerImplProvider{dataManager: dataManager}, stopCh))

	candidateManager.AddImpl("baremetal", NewCandidateManagerImpl(
		&BaremetalCandidateManagerImplProvider{dataManager: dataManager}, stopCh))

	return candidateManager
}

func (cm *CandidateManager) Run() {

	for _, impl := range cm.impls {
		impl.Run()
	}
}

func (cm *CandidateManager) GetData(name string) ([]interface{}, error) {
	cache, err := cm.dataManager.DBCacheGroup.Get(name)
	if err != nil {
		return nil, err
	}

	cache.WaitForReady()
	return cache.List(), nil
}

func (cm *CandidateManager) Reload(resType string, candidateIds []string) (
	[]interface{}, error) {

	if len(candidateIds) == 0 {
		return []interface{}{}, nil
	}

	impl, err := cm.getImpl(resType)
	if err != nil {
		return nil, err
	}

	return impl.Reload(candidateIds)
}

func (cm *CandidateManager) ReloadAll(resType string) ([]interface{}, error) {
	impl, err := cm.getImpl(resType)
	if err != nil {
		return nil, err
	}

	return impl.ReloadAll()
}

func (cm *CandidateManager) SetCandidatesDirty(scs []*core.SelectedCandidate) {
	for _, sc := range scs {
		cm.dirtyPool.Add(sc, uint64(sc.Count))
	}
}

func (cm *CandidateManager) CleanDirtyCandidatesOnce(keys []string) {
	for _, key := range keys {
		cm.dirtyPool.DeleteByKey(key)
	}
}

func ToHostCandidate(c interface{}) (*candidatecache.HostDesc, error) {
	h, ok := c.(*candidatecache.HostDesc)
	if !ok {
		return nil, fmt.Errorf("%#v can't convert to *candidatecache.HostDesc")
	}
	return h, nil
}

func ToHostCandidates(cs []interface{}) ([]*candidatecache.HostDesc, error) {
	hs := make([]*candidatecache.HostDesc, 0)
	for _, c := range cs {
		h, err := ToHostCandidate(c)
		if err != nil {
			return nil, err
		}
		hs = append(hs, h)
	}
	return hs, nil
}
