package options

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

var (
	NameSyncResources stringutils2.SSortedStrings

	SyncPurgeRemovedResources stringutils2.SSortedStrings
)

func InitNameSyncResources() {
	NameSyncResources = stringutils2.NewSortedStrings(Options.NameSyncResources)
	log.Infof("NameSyncResources: %s", NameSyncResources)

	SyncPurgeRemovedResources = stringutils2.NewSortedStrings(Options.SyncPurgeRemovedResources)
	log.Infof("SyncPurgeRemovedResources: %s", SyncPurgeRemovedResources)
}
