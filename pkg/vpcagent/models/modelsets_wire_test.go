package models

import (
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/reflectutils"
)

var nodeRefPattern = regexp.MustCompile(`<[0-9]+>`)

// expectedModelSetKeys pins the wire format of ModelSets: the json key of every
// model set, sorted.  Marshal and Unmarshal both derive the key through
// reflectutils.Info.MarshalName(), and Unmarshal silently drops an unmatched
// key, so a renamed field would drop a whole model set without an error.  Keep
// this list in sync with the json tags of ModelSets.
var expectedModelSetKeys = []string{
	"dns_records",
	"dns_zones",
	"elasticips",
	"groupguests",
	"groupnetworks",
	"groups",
	"guestnetworks",
	"guestnetworksecgroups",
	"guests",
	"guestsecgroups",
	"hosts",
	"loadbalancer_acls",
	"loadbalancer_listeners",
	"loadbalancer_networks",
	"network_addresses",
	"networks",
	"route_tables",
	"security_group_rules",
	"security_groups",
	"vpcs",
	"wires",
}

func buildModelSetsForWireTest() *ModelSets {
	mss := NewModelSets()

	vpc := &Vpc{}
	vpc.Id = "vpc1"
	vpc.Name = "vpc1"
	mss.Vpcs[vpc.Id] = vpc

	wire := &Wire{}
	wire.Id = "wire1"
	wire.Name = "wire1"
	wire.VpcId = "vpc1"
	mss.Wires[wire.Id] = wire

	rt := &RouteTable{}
	rt.Id = "rt1"
	rt.Name = "rt1"
	rt.VpcId = "vpc1"
	mss.RouteTables[rt.Id] = rt

	net := &Network{}
	net.Id = "net1"
	net.Name = "net1"
	net.WireId = "wire1"
	mss.Networks[net.Id] = net

	zone := &DnsZone{}
	zone.Id = "zone1"
	zone.Name = "zone1"
	mss.DnsZones[zone.Id] = zone

	rec := &DnsRecord{}
	rec.Id = "rec1"
	rec.Name = "rec1"
	rec.DnsZoneId = "zone1"
	mss.DnsRecords[rec.Id] = rec

	host := &Host{}
	host.Id = "host1"
	host.Name = "host1"
	mss.Hosts[host.Id] = host

	guest := &Guest{}
	guest.Id = "guest1"
	guest.Name = "guest1"
	guest.HostId = "host1"
	mss.Guests[guest.Id] = guest

	gn := &Guestnetwork{}
	gn.RowId = 1
	gn.GuestId = "guest1"
	gn.NetworkId = "net1"
	gn.IpAddr = "10.0.0.2"
	mss.Guestnetworks["1"] = gn

	return mss
}

// TestModelSetsKeyList checks the wire key of every model set, and that each key
// is matched back to exactly one model set by Unmarshal.
func TestModelSetsKeyList(t *testing.T) {
	fields := reflectutils.FetchStructFieldValueSet(reflect.ValueOf(ModelSets{}))
	keys := make([]string, 0, len(fields))
	for i := range fields {
		keys = append(keys, fields[i].Info.MarshalName())
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, expectedModelSetKeys) {
		t.Fatalf("model set keys changed:\n got: %v\nwant: %v", keys, expectedModelSetKeys)
	}
}

// TestModelSetsStatsKeyList checks that ModelSetsStats carries the same wire
// keys as ModelSets, so that /modelsets/stats stays consistent with the model
// sets it counts.
func TestModelSetsStatsKeyList(t *testing.T) {
	fields := reflectutils.FetchStructFieldValueSet(reflect.ValueOf(ModelSetsStats{}))
	keys := make([]string, 0, len(fields))
	for i := range fields {
		keys = append(keys, fields[i].Info.MarshalName())
	}
	sort.Strings(keys)
	if !reflect.DeepEqual(keys, expectedModelSetKeys) {
		t.Fatalf("model sets stats keys changed:\n got: %v\nwant: %v", keys, expectedModelSetKeys)
	}
}

func TestModelSetsKeysMatchedByUnmarshal(t *testing.T) {
	for _, k := range expectedModelSetKeys {
		js := `{"` + k + `":{"probe":{"id":"probe","name":"probe"}}}`
		obj, err := jsonutils.Parse([]byte(js))
		if err != nil {
			t.Fatalf("parse probe for key %s: %v", k, err)
		}
		mss := NewModelSets()
		if err := obj.Unmarshal(mss); err != nil {
			t.Fatalf("unmarshal probe for key %s: %v", k, err)
		}
		matched := 0
		for _, ms := range mss.ModelSetList() {
			if reflect.ValueOf(ms).Len() > 0 {
				matched++
			}
		}
		if matched != 1 {
			t.Fatalf("key %q matched %d model sets, want exactly 1", k, matched)
		}
	}
}

// TestModelSetsFlatRoundTrip joins a ModelSets, checks that the marshaled form
// is flat (no join link, no node reference, so that a plain Parse can read it
// back), and that the receiver side restores the links through ApplyUpdates.
func TestModelSetsFlatRoundTrip(t *testing.T) {
	mss := buildModelSetsForWireTest()
	mss.join()

	if mss.Wires["wire1"].Vpc == nil {
		t.Fatal("join did not link wire->vpc")
	}
	if mss.RouteTables["rt1"].Vpc == nil {
		t.Fatal("join did not link routetable->vpc")
	}
	if mss.DnsRecords["rec1"].DnsZone == nil {
		t.Fatal("join did not link dnsrecord->dnszone")
	}
	if mss.Networks["net1"].Wire == nil {
		t.Fatal("join did not link network->wire")
	}

	s := jsonutils.Marshal(mss).String()

	if strings.Contains(s, "___jnid_") {
		t.Fatalf("marshaled output contains node id marker ___jnid_")
	}
	if nodeRefPattern.MatchString(s) {
		t.Fatalf("marshaled output contains a node reference <N>")
	}

	obj, err := jsonutils.Parse([]byte(s))
	if err != nil {
		t.Fatalf("plain Parse of marshaled modelsets failed: %v", err)
	}

	// flatness: no join link carried over the wire
	for _, c := range []struct {
		path []string
		keys []string
	}{
		{[]string{"wires", "wire1"}, []string{"vpc", "Vpc"}},
		{[]string{"route_tables", "rt1"}, []string{"vpc", "Vpc"}},
		{[]string{"dns_records", "rec1"}, []string{"dns_zone", "DnsZone"}},
		{[]string{"dns_zones", "zone1"}, []string{"records", "Records"}},
		{[]string{"networks", "net1"}, []string{"wire", "Wire", "vpc", "Vpc"}},
		{[]string{"guests", "guest1"}, []string{"host", "Host"}},
		{[]string{"guestnetworks", "1"}, []string{"guest", "Guest", "network", "Network"}},
	} {
		o, err := obj.Get(c.path...)
		if err != nil {
			t.Fatalf("get %v: %v", c.path, err)
		}
		for _, k := range c.keys {
			if o.Contains(k) {
				t.Fatalf("%v still carries join link %q", c.path, k)
			}
		}
	}

	// receiver side: parse into a fresh set, apply updates, join
	mssNews := NewModelSets()
	if err := obj.Unmarshal(mssNews); err != nil {
		t.Fatalf("unmarshal into ModelSets: %v", err)
	}
	if len(mssNews.Vpcs) != 1 || len(mssNews.Wires) != 1 || len(mssNews.RouteTables) != 1 ||
		len(mssNews.Networks) != 1 || len(mssNews.DnsZones) != 1 || len(mssNews.DnsRecords) != 1 ||
		len(mssNews.Guests) != 1 || len(mssNews.Guestnetworks) != 1 {
		t.Fatalf("round trip lost members: %s", mssNews.Stats().Dump())
	}
	if mssNews.Wires["wire1"] == nil || mssNews.Wires["wire1"].VpcId != "vpc1" {
		t.Fatal("wire1 did not survive the round trip")
	}
	if mssNews.DnsRecords["rec1"] == nil || mssNews.DnsRecords["rec1"].DnsZoneId != "zone1" {
		t.Fatal("rec1 did not survive the round trip")
	}

	local := NewModelSets()
	r := local.ApplyUpdates(mssNews)
	if !r.Changed {
		t.Fatal("ApplyUpdates reported no change")
	}
	if !r.Correct {
		t.Fatal("ApplyUpdates join failed")
	}
	if local.Wires["wire1"] == nil || local.Wires["wire1"].Vpc == nil {
		t.Fatal("wire->vpc link not restored after ApplyUpdates")
	}
	if local.RouteTables["rt1"] == nil || local.RouteTables["rt1"].Vpc == nil {
		t.Fatal("routetable->vpc link not restored after ApplyUpdates")
	}
	if local.DnsRecords["rec1"] == nil || local.DnsRecords["rec1"].DnsZone == nil {
		t.Fatal("dnsrecord->dnszone link not restored after ApplyUpdates")
	}
	if local.Guestnetworks["1"] == nil || local.Guestnetworks["1"].Network == nil {
		t.Fatal("guestnetwork->network link not restored after ApplyUpdates")
	}
	if local.Guestnetworks["1"].Guest == nil || local.Guests["guest1"].Host == nil {
		t.Fatal("guestnetwork->guest / guest->host link not restored after ApplyUpdates")
	}
}
