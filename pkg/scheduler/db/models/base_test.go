package models

import (
	"flag"
	"fmt"
	"testing"
)

var (
	dialect = flag.String("db-dialect", "mysql", "db dialect")
	dbURL   = flag.String("db-url", "root:root@tcp(127.0.0.1:3306)/yunioncloud?charset=utf8&parseTime=True", "db url")
)

func init() {
	flag.Parse()
	err := Init(*dialect, *dbURL)
	if err != nil {
		panic(fmt.Errorf("Test init error: %v", err))
	}
}

func TestQuery(t *testing.T) {
	ids, err := AllIDs(Guests)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v: , length: %d", ids, len(ids))

	objs, err := All(Guests)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%v: , length: %d", objs[1], len(objs))
}

func TestQueryIn(t *testing.T) {
	ids := []string{"000ea33f-f751-4f7f-85ef-958676a5e78b", "000f5af0-ee2b-4678-a7ce-a93987f2a87d"}
	bms, err := FetchByIDs(Baremetals, ids)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Bms: %v, length: %d", bms, len(bms))
}

func TestFetchByHostIDs(t *testing.T) {
	ids := []string{"7916bd54-40b5-4465-842c-832e4e42313f"}
	objs, err := FetchByHostIDs(Guests, ids)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Guests: %v, length: %d", objs, len(objs))
}

func TestHostStorage(t *testing.T) {
	ss, err := All(HostStorages)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("HostStorages: %v, length: %d", ss, len(ss))
}

func TestStorage(t *testing.T) {
	ss, err := All(Storages)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range ss {
		storage := s.(*Storage)
		if storage.ZoneID != "" {
			t.Logf("Storage: %#v", storage)
		}
	}
}

func TestGroup(t *testing.T) {
	groups, err := All(Groups)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("groups: %v, len: %d", groups, len(groups))
}

func TestGroupGuest(t *testing.T) {
	groups, err := All(GroupGuests)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("group guests: %v, len: %d", groups[0], len(groups))
}

func TestMetadata(t *testing.T) {
	metadatas, err := AllWithDeleted(Metadatas)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("metadata: %v, len: %d", metadatas[0], len(metadatas))
}

func TestIsolatedDev(t *testing.T) {
	devs, err := All(IsolatedDevices)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("IsolatedDevices: %+v, len: %d", devs[0], len(devs))
}

func TestDisk(t *testing.T) {
	disks, err := All(Disks)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Disks: %v, len: %d", disks[0], len(disks))
	capas, err := GetStorageCapacities([]string{"d0205a6a-b8aa-4365-ba5e-1003104006a8"})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Capacities: %v, len: %d", capas, len(capas))
}

func TestGuestTenant(t *testing.T) {
	hostids := []string{"7916bd54-40b5-4465-842c-832e4e42313f"}
	ts, err := ResidentTenantsInHosts(hostids)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("tenants: %v, len: %d", ts, len(ts))
}

func TestFetchMetadatas(t *testing.T) {
	hostids := []string{"7916bd54-40b5-4465-842c-832e4e42313f"}
	serverids := []string{"fffd63c7-b0ef-446b-bfbf-ad05e1cefe2a"}
	hostMetadataNames := []string{"dynamic_load_cpu_percent", "dynamic_load_io_util",
		"enable_sriov", "bridge_driver"}
	hostMetadataNames = append(hostMetadataNames, HostExtraFeature...)
	hostMetadatas, err := FetchMetadatas(HostResourceName, hostids, hostMetadataNames)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("hostMetadatas: %v", hostMetadatas)
	guestMetadataNames := []string{"app_tags"}
	guestMetadatas, err := FetchMetadatas(GuestResourceName, serverids, guestMetadataNames)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("guestMetadatas: %v", guestMetadatas)
}

func TestGuestDisk(t *testing.T) {
	disks, err := All(GuestDisks)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Disks: %v, len: %d", disks[0], len(disks))
	gst, err := FetchByID(Guests, "b4438e03-c6c2-4f88-8b95-11efea0300c4")
	if err != nil {
		t.Fatal(err)
	}
	size, err := gst.(*Guest).DiskSize()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("DiskSize: %d", size)
}

func BenchmarkQueryUseScanRow(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, err := All(Guests)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func allTest(r Resourcer) (interface{}, error) {
	cond := map[string]interface{}{
		"deleted": false,
	}
	objs := r.Models()
	if err := r.DB().Where(cond).Find(objs).Error; err != nil {
		return nil, err
	}
	return objs, nil
}

func BenchmarkQueryUseSlice(b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, err := allTest(Guests)
		if err != nil {
			b.Fatal(err)
		}
	}
}
