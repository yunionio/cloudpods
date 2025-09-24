package aws

import (
	"container/list"
	"context"
	"net"
	"net/http"
	"sync"
	"time"
)

var DnsCacheTransport = &http.Transport{
	Proxy: http.ProxyFromEnvironment,
	DialContext: DnsCacheTransportDialContext(&net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}, NewDnsResolver(100)),
	ForceAttemptHTTP2:     true,
	MaxIdleConns:          100,
	IdleConnTimeout:       90 * time.Second,
	TLSHandshakeTimeout:   10 * time.Second,
	ExpectContinueTimeout: 1 * time.Second,
}

func DnsCacheTransportDialContext(dialer *net.Dialer, resolver *DnsResolver) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, address string) (net.Conn, error) {
		conn, err := dialer.DialContext(ctx, network, address)
		if err != nil {
			remoteAddr, exists := resolver.cache.Get(address)
			if exists {
				return dialer.DialContext(ctx, network, remoteAddr)
			}
			return conn, err
		}

		if conn.RemoteAddr().String() != "" {
			resolver.cache.Set(address, conn.RemoteAddr().String())
		}

		return conn, err
	}
}

type DnsResolver struct {
	cache *FIFOCache
}

// NewDnsResolver 创建一个新的 DNS 解析器，使用固定长度的 FIFO 缓存
func NewDnsResolver(maxSize int) *DnsResolver {
	return &DnsResolver{
		cache: NewFIFOCache(maxSize),
	}
}

// FIFOCache 实现一个固定大小的并发安全FIFO（先进先出）缓存
// 当缓存满时，添加新条目会自动淘汰最早插入的条目
type FIFOCache struct {
	maxSize int               // 缓存的最大容量
	cache   map[string]string // 存储键值对的映射
	keys    *list.List        // 使用链表维护键的插入顺序（FIFO）
	rwMutex sync.RWMutex      // 读写锁，保证并发安全
}

// NewFIFOCache 创建并返回一个新的FIFOCache实例
// 参数:
//
//	maxSize: 缓存的最大容量，必须大于0，否则默认为100
//
// 返回值:
//
//	*FIFOCache: 新创建的缓存实例
func NewFIFOCache(maxSize int) *FIFOCache {
	if maxSize < 1 {
		maxSize = 100
	}
	return &FIFOCache{
		maxSize: maxSize,
		cache:   make(map[string]string),
		keys:    list.New(),
	}
}

// Set 添加或更新一个键值对到缓存中
// 如果缓存已满且键不存在，会淘汰最早插入的条目
// 参数:
//
//	key: 要添加或更新的键
//	value: 要添加或更新的值
//
// 返回值:
//
//	string: 如果键已存在，返回被替换的旧值；否则返回空字符串
//	bool: 表示是否替换了现有值（true表示替换，false表示新增）
func (c *FIFOCache) Set(key, value string) (string, bool) {
	c.rwMutex.Lock()
	defer c.rwMutex.Unlock()

	// 检查键是否已存在
	oldValue, exists := c.cache[key]
	if exists {
		// 更新现有键的值
		c.cache[key] = value
		return oldValue, true
	}

	// 缓存已满，淘汰最旧的条目
	if c.keys.Len() >= c.maxSize {
		oldest := c.keys.Front()
		if oldest != nil {
			oldestKey := oldest.Value.(string)
			delete(c.cache, oldestKey)
			c.keys.Remove(oldest)
		}
	}

	// 添加新条目
	c.cache[key] = value
	c.keys.PushBack(key)
	return "", false
}

// Get 从缓存中获取指定键对应的值
// 参数:
//
//	key: 要查找的键
//
// 返回值:
//
//	string: 找到的值（如果键不存在则返回空字符串）
//	bool: 表示键是否存在（true表示存在，false表示不存在）
func (c *FIFOCache) Get(key string) (string, bool) {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()

	value, exists := c.cache[key]
	return value, exists
}

// Size 返回缓存中当前存储的条目数量
// 返回值:
//
//	int: 当前缓存中的条目数量
func (c *FIFOCache) Size() int {
	c.rwMutex.RLock()
	defer c.rwMutex.RUnlock()
	return len(c.cache)
}

// GetMaxSize 返回缓存的最大容量
// 返回值:
//
//	int: 缓存的最大容量
func (c *FIFOCache) GetMaxSize() int {
	return c.maxSize
}
