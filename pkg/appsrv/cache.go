package appsrv

import (
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"sync"
)

type cacheNode struct {
	key   string
	value interface{}
}

type Cache struct {
	table []cacheNode
	lock  *sync.Mutex
	size  uint32
}

func NewCache(size uint32) *Cache {
	ca := &Cache{table: make([]cacheNode, size), lock: &sync.Mutex{}, size: size}
	return ca
}

const (
	HASH_ALG_MD5 int = iota
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
	for _, alg := range []int{HASH_ALG_MD5, HASH_ALG_SHA1, HASH_ALG_SHA256} {
		idx = checksum(alg, key) % c.size
		if c.table[idx].key == key {
			return true, idx
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

func (c *Cache) Set(key string, val interface{}) {
	find, idx := c.find(key)
	if find {
		c.table[idx].value = val
	} else {
		c.table[idx].key = key
		c.table[idx].value = val
	}
}

func (c *Cache) AtomicSet(key string, val interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.Set(key, val)
}
