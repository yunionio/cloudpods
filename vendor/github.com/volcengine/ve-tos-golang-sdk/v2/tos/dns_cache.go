package tos

import (
	"container/heap"
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	DefaultCacheCap = 100
	VolceHostSuffix = "volces.com"
	HostSplitSep    = "."
	HostSplitLength = 4
)

type cacheItem struct {
	host      string
	ipList    []string
	expireAt  time.Time
	heapIndex int
}
type priorityQueue []*cacheItem

func (p priorityQueue) Len() int {
	return len(p)
}

func (p priorityQueue) Peek() *cacheItem {
	if p.Len() > 0 {
		return p[0]
	}
	return nil
}

func (p priorityQueue) Less(i, j int) bool {
	return p[i].expireAt.Before(p[j].expireAt)
}

func (p priorityQueue) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
	p[i].heapIndex = i
	p[j].heapIndex = j
}

func (p *priorityQueue) Push(x interface{}) {
	n := len(*p)
	item := x.(*cacheItem)
	item.heapIndex = n
	*p = append(*p, item)
}

func (p *priorityQueue) Pop() interface{} {
	old := *p
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.heapIndex = -1
	*p = old[0 : n-1]
	return item
}

type cache struct {
	lock       sync.RWMutex
	heap       *priorityQueue
	cleanTime  time.Time
	data       map[string]cacheItem
	expiration time.Duration
}

func (c *cache) Remove(key string, removeIp string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	data, ok := c.data[key]
	if !ok {
		return
	}
	value := make([]string, 0, len(data.ipList))
	for _, ip := range data.ipList {
		if ip == removeIp {
			continue
		}
		value = append(value, ip)
	}

	// 没有有效的 IP 将缓存删除
	if len(value) == 0 {
		delete(c.data, key)
		return
	}

	data.ipList = value
	c.data[key] = data

}

func (c *cache) Get(key string) ([]string, bool) {
	c.lock.RLock()
	data, ok := c.data[key]
	c.lock.RUnlock()
	if !ok {
		return nil, false
	}
	if data.expireAt.Before(time.Now()) {
		return nil, false
	}
	return data.ipList, true
}

func (c *cache) cleanCache() {
	c.cleanTime = time.Now().Add(c.expiration)
	maxCleanCount := 5
	for i := 0; i < maxCleanCount; i++ {
		item := c.heap.Peek()
		if item == nil {
			return
		}

		if item.expireAt.Before(time.Now()) {
			heap.Pop(c.heap)
			data, ok := c.data[item.host]
			if ok && data.expireAt == item.expireAt {
				delete(c.data, item.host)
			}
		} else {
			return
		}
	}
}

func (c *cache) Put(key string, ipList []string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	item := cacheItem{
		ipList:   ipList,
		expireAt: time.Now().Add(c.expiration),
		host:     key,
	}
	c.data[key] = item
	heap.Push(c.heap, &item)

	// 大于 Cap
	if c.heap.Len() > DefaultCacheCap {
		item := heap.Pop(c.heap).(*cacheItem)
		if item == nil {
			return
		}
		data, ok := c.data[item.host]
		if ok && data.expireAt == item.expireAt {
			delete(c.data, item.host)
		}
	}

	if time.Now().After(c.cleanTime) {
		c.cleanCache()
	}
}

type resolver struct {
	cache *cache
}

func newResolver(expiration time.Duration) *resolver {
	pq := make(priorityQueue, 0)
	return &resolver{cache: &cache{
		heap:       &pq,
		cleanTime:  time.Now().Add(expiration),
		data:       make(map[string]cacheItem),
		expiration: expiration,
	}}
}

func ipToStringList(ips []net.IP) []string {
	res := make([]string, len(ips))
	for i, ip := range ips {
		res[i] = ip.String()
	}
	return res
}

func wrappedHost(host string) string {
	if !strings.HasSuffix(host, VolceHostSuffix) {
		return host
	}
	hostSplit := strings.Split(host, HostSplitSep)
	if len(hostSplit) != HostSplitLength {
		return host
	}
	return strings.Join(hostSplit[1:], HostSplitSep)

}

func (r *resolver) GetIpList(ctx context.Context, host string) ([]string, error) {

	ipList, ok := r.cache.Get(wrappedHost(host))
	if ok {
		return ipList, nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	ipsStr := ipToStringList(ips)
	r.cache.Put(wrappedHost(host), ipsStr)

	return ipsStr, nil
}

func (r *resolver) Remove(host string, ip string) {
	r.cache.Remove(wrappedHost(host), ip)
}
