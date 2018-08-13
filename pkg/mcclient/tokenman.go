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
