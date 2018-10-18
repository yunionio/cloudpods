package models

import (
	"yunion.io/x/onecloud/pkg/scheduler/db"
)

const (
	hostsTable    = "hosts_tbl"
	clustersTable = "clusters_tbl"
	zonesTable    = "zones_tbl"
	guestsTable   = "guests_tbl"

	baremetalsTable        = "baremetals_tbl"
	baremetalAgentsTable   = "baremetalagents_tbl"
	baremetalNetworksTable = "baremetalnetworks_tbl"

	storageTable     = "storages_tbl"
	hostStorageTable = "hoststorages_tbl"

	groupGuestTable    = "guestgroups_tbl"
	groupTable         = "groups_tbl"
	groupNetworksTable = "groupnetworks_tbl"

	metadataTable = "metadata_tbl"

	isolatedDeviceTable = "isolated_devices_tbl"

	disksTable         = "disks_tbl"
	guestDiskTable     = "guestdisks_tbl"
	guestNetworksTable = "guestnetworks_tbl"

	aggregatesTable     = "aggregates_tbl"
	aggregateHostsTable = "aggregate_hosts_tbl"

	networksTable      = "networks_tbl"
	netinterfacesTable = "netinterfaces_tbl"

	wiresTable     = "wires_tbl"
	hostWiresTable = "hostwires_tbl"

	reserveDipsTable = "reservedips_tbl"

	dynamicschedtagTable = "dynamicschedtags_tbl"
	cloudproviderTable   = "cloudproviders_tbl"
)

var (
	Hosts     Resourcer
	HostWires Resourcer

	Clusters       Resourcer
	Zones          Resourcer
	Guests         Resourcer
	CloudProviders Resourcer

	Baremetals        Resourcer
	BaremetalAgents   Resourcer
	BaremetalNetworks Resourcer

	Storages     Resourcer
	HostStorages Resourcer

	GroupGuests   Resourcer
	Groups        Resourcer
	GroupNetworks Resourcer

	Metadatas Resourcer

	IsolatedDevices Resourcer
	Disks           Resourcer

	GuestDisks    Resourcer
	GuestNetworks Resourcer

	Aggregates       Resourcer
	AggregateHosts   Resourcer
	Dynamicschedtags Resourcer

	Networks      Resourcer
	NetInterfaces Resourcer

	Wires Resourcer

	ReserveDipsNerworks Resourcer
)

func Init(dialect string, args ...interface{}) error {
	err := db.Init(dialect, args...)
	if err != nil {
		return err
	}
	Hosts, _ = NewHostResource(db.DB)
	Clusters, _ = NewClusterResource(db.DB)
	Zones, _ = NewZoneResource(db.DB)
	Guests, _ = NewGuestResource(db.DB)
	CloudProviders, _ = NewCloudproviderResource(db.DB)

	Baremetals, _ = NewBaremetalResource(db.DB)
	BaremetalAgents, _ = NewBaremetalAgentResource(db.DB)

	Storages, _ = NewStorageResource(db.DB)
	HostStorages, _ = NewHostStorageResource(db.DB)

	Groups, _ = NewGroupResource(db.DB)
	GroupGuests, _ = NewGroupGuestResource(db.DB)

	Metadatas, _ = NewMetadataResource(db.DB)

	IsolatedDevices, _ = NewIsolatedDeviceResource(db.DB)

	Disks, _ = NewDiskResource(db.DB)
	GuestDisks, _ = NewGuestDiskResource(db.DB)

	Aggregates, _ = NewAggregateResource(db.DB)
	AggregateHosts, _ = NewAggregateHostResource(db.DB)
	Dynamicschedtags, _ = NewDynaimcschedtagResource(db.DB)

	Networks, _ = NewNetworksResource(db.DB)
	NetInterfaces, _ = NewNetInterfacesResource(db.DB)
	Wires, _ = NewWiresResource(db.DB)
	HostWires, _ = NewHostWiresResource(db.DB)
	GuestNetworks, _ = NewGuestNetworksResource(db.DB)
	GroupNetworks, _ = NewGroupNetworksResource(db.DB)
	BaremetalNetworks, _ = NewBaremetalNetworksResource(db.DB)
	ReserveDipsNerworks, _ = NewReserveDipsNetworksResource(db.DB)
	return nil
}

func DBValid() bool {
	if db.DB == nil {
		return false
	}
	return true
}
