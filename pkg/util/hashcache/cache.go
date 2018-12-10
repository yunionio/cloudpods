package hashcache

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"sync"
	"time"
)

type cacheNode struct {
	key    string
	expire time.Time
	value  interface{}
}

type Cache struct {
	table      []cacheNode
	lock       *sync.Mutex
	size       uint32
	defaultTtl time.Duration
}

func NewCache(size uint32, defaultTTL time.Duration) *Cache {
	ca := &Cache{table: make([]cacheNode, size), lock: &sync.Mutex{}, size: size, defaultTtl: defaultTTL}
	return ca
}

const (
	HASH_ALG_MD5    int = iota
	HASH_ALG_SHA1
	HASH_ALG_SHA256
)

func bytes2int(b []byte) uint32 {
	return ((uint32(b[0]) << 24) | (uint32(b[1]) << 16) | (uint32(b[2]) << 8) | (uint32(b[3])))
}

func checksum(alg int, key string) uint32 {
	var hash uint32 = 0
	switch alg {
	case HASH_ALG_MD5:
		v := md5.Sum([]byte(key))
		hash = bytes2int(v[0:4])
	case HASH_ALG_SHA1:
		v := sha1.Sum([]byte(key))
		hash = bytes2int(v[0:4])
	case HASH_ALG_SHA256:
		v := sha256.Sum224([]byte(key))
		hash = bytes2int(v[0:4])
	}
	return hash
}

func (c *Cache) find(key string) (bool, uint32) {
	var idx uint32
	now := time.Now()
	for _, alg := range []int{HASH_ALG_MD5, HASH_ALG_SHA1, HASH_ALG_SHA256} {
		idx = checksum(alg, key) % c.size
		if c.table[idx].key == key {
			if c.table[idx].expire.IsZero() || c.table[idx].expire.After(now) {
				return true, idx
			}
			break
		}
	}
	return false, idx
}

func (c *Cache) Get(key string) interface{} {
	find, idx := c.find(key)
	if find {
		return c.table[idx].value
	}
	return nil
}

func (c *Cache) AtomicGet(key string) interface{} {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.Get(key)
}

func (c *Cache) Set(key string, val interface{}, expire ...time.Time) {
	find, idx := c.find(key)
	if !find {
		c.table[idx].key = key
	}
	c.table[idx].value = val
	if len(expire) > 0 && ! expire[0].IsZero() {
		c.table[idx].expire = expire[0]
	} else if c.defaultTtl > time.Millisecond {
		c.table[idx].expire = time.Now().Add(c.defaultTtl)
	} else {
		c.table[idx].expire = time.Time{}
	}
}

func (c *Cache) AtomicSet(key string, val interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.Set(key, val)
}
