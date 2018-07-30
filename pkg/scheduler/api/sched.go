package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/bitly/go-simplejson"

	o "github.com/yunionio/onecloud/cmd/scheduler/options"
	"github.com/yunionio/log"
	"github.com/yunionio/pkg/utils"
)

type Meta map[string]string

type Disk struct {
	Backend         string  `json:"backend"`
	ImageID         string  `json:"image_id"`
	Fs              *string `json:"fs"`
	Os              string  `json:"os"`
	OSDistribution  string  `json:"os_distribution"`
	OsVersion       string  `json:"os_version"`
	Format          string  `json:"format"`
	MountPoint      *string `json:"mountpoint"`
	Driver          *string `json:"driver"`
	Cache           *string `json:"cache"`
	ImageDiskFormat string  `json:"image_disk_format"`
	Size            int64   `json:"size"`
	Storage         *string `json:"storage"`
}

type Network struct {
	Idx      string `json:"idx"`
	TenantId string `json:"tenant_id"`
	Private  bool   `json:"private"`
	Ports    int64  `json:"ports"`
	Exit     bool   `json:"exit"`
	Wire     string `json:"wire"`
	Mac      string `json:"mac"`
	Address  string `json:"address"`
	Address6 string `json:"address6"`
	Driver   string `json:"driver"`
	BwLimit  int64  `json:"bw_limit"`
	Vip      bool   `json:"vip"`
	Reserved bool   `json:"reserved"`
}

type IsolatedDevice struct {
	ID     string `json:"id"`
	Type   string `json:"dev_type"`
	Model  string `json:"model"`
	Vendor string `json:"vendor"`
}

type ForGuest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Aggregate struct {
	Idx      string `json:"idx"`
	Strategy string `json:"strategy"`
}

type GroupRelation struct {
	GroupID  string `json:"group_id"`
	Strategy string `json:"strategy"`
	Scope    string `json:"scope"`
}

type SchedData struct {
	Tag             string            `json:"tag"`
	Type            string            `json:"type"`
	IsContainer     bool              `json:"is_container"`
	Count           int64             `json:"count"`
	ZoneID          string            `json:"zone_id"`
	PoolID          string            `json:"pool_id"`
	HostID          string            `json:"host_id"`
	Candidates      []string          `json:"candidates"`
	OwnerTenantID   string            `json:"owner_tenant_id"`
	OwnerUserID     string            `json:"owner_user_id"`
	VMEMSize        int64             `json:"vmem_size"`
	VCPUCount       int64             `json:"vcpu_count"`
	Disks           []*Disk           `json:"disks"`
	Name            string            `json:"name"`
	Networks        []*Network        `json:"networks"`
	IsolatedDevices []*IsolatedDevice `json:"isolated_devices"`
	Aggregates      []Aggregate       `json:"aggregate_stategy"`
	Meta            Meta              `json:"__meta__"`
	ForGuests       []*ForGuest       `json:"for_guests"`
	GuestStatus     string            `json:"guest_status"`
	Hypervisor      string            `json:"hypervisor"`

	// VM
	Groups         []string        `json:"group"`
	GroupRelations []GroupRelation `json:"group_relations"`

	// baremental
	BaremetalDiskConfigs []*BaremetalDiskConfig `json:"baremetal_disk_config"`
}

func NewSchedData(sjson *simplejson.Json, count int64, byTest bool) (*SchedData, error) {
	data := new(SchedData)

	if tag, ok := sjson.CheckGet("tag"); ok {
		if str, err := tag.String(); err == nil {
			data.Tag = str
		}
	}

	if zoneID, ok := sjson.CheckGet("prefer_zone_id"); ok {
		if str, err := zoneID.String(); err == nil {
			data.ZoneID = str
		}
	} else if zoneID, ok := sjson.CheckGet("prefer_zone"); ok {
		if str, err := zoneID.String(); err == nil {
			data.ZoneID = str
		}
	}

	if poolID, ok := sjson.CheckGet("prefer_pool_id"); ok {
		if str, err := poolID.String(); err == nil {
			data.PoolID = str
		}
	}

	candidates := make([]string, 0)

	if hostID, ok := sjson.CheckGet("prefer_host_id"); ok {
		if str, err := hostID.String(); err == nil {
			candidates = append(candidates, str)
		}
	} else if hostID, ok := sjson.CheckGet("prefer_host"); ok {
		if str, err := hostID.String(); err == nil {
			candidates = append(candidates, str)
		}
	}

	if baremetalID, ok := sjson.CheckGet("prefer_baremetal_id"); ok {
		if str, err := baremetalID.String(); err == nil {
			candidates = append(candidates, str)
		}
	} else if baremetalID, ok := sjson.CheckGet("prefer_baremetal_id"); ok {
		if str, err := baremetalID.String(); err == nil {
			candidates = append(candidates, str)
		}
	}

	data.Candidates = candidates

	err := data.reviseSchedType(sjson)
	if err != nil {
		return nil, err
	}

	data.reviseSchedHypervisor(sjson)

	data.VMEMSize = sjson.Get("vmem_size").MustInt64()
	data.OwnerTenantID = sjson.Get("owner_tenant_id").MustString()
	data.OwnerUserID = sjson.Get("owner_user_id").MustString()
	data.VCPUCount = sjson.Get("vcpu_count").MustInt64()
	data.Name = sjson.Get("name").MustString()
	data.Count = count
	data.GuestStatus = sjson.Get("guest_status").MustString()
	data.Meta = map[string]string{}
	data.HostID = sjson.Get("host_id").MustString()

	for key, value := range sjson.Get("__meta__").MustMap() {
		data.Meta[key] = fmt.Sprintf("%v", value)
	}

	if err := data.fillNetworksInfo(sjson); err != nil {
		return nil, err
	}

	if err := data.fillIsolatedDeviceInfo(sjson); err != nil {
		return nil, err
	}

	if err := data.fillDisksInfo(sjson, byTest); err != nil {
		return nil, err
	}

	data.fillForGuests(sjson)

	if err := data.fillAggregates(sjson, byTest); err != nil {
		return nil, err
	}

	if err := data.fillGroupRelations(sjson); err != nil {
		return nil, err
	}

	if err := data.fillBaremetalDiskConfig(sjson); err != nil {
		return nil, err
	}

	if err := data.fillGroups(sjson); err != nil {
		return nil, err
	}

	data.reviseSchedData()

	return data, nil
}

func (s *SchedData) reviseSchedHypervisor(sjson *simplejson.Json) {
	var finalHypervisor string
	switch hypervisor := sjson.Get("hypervisor").MustString(); hypervisor {
	case "", SchedTypeKvm:
		finalHypervisor = HostHypervisorForKvm
	default:
		finalHypervisor = hypervisor
	}
	s.Hypervisor = finalHypervisor
}

func (s *SchedData) reviseSchedType(sjson *simplejson.Json) error {
	var bodyType string
	if reqType, ok := sjson.CheckGet("type"); ok {
		bodyType = reqType.MustString()
	} else {
		bodyType = HostTypeHost
	}

	switch bodyType {
	case "", HostTypeHost, SchedTypeGuest, SchedTypeEsxi, SchedTypeKvm, SchedTypeHyperV:
		s.Type = SchedTypeGuest

	case SchedTypeContainer:
		s.Type = SchedTypeGuest
		s.IsContainer = true

	case HostTypeBaremetal:
		s.Type = SchedTypeBaremetal

	default:
		return fmt.Errorf("Sched current type=%s not support", bodyType)
	}
	return nil
}

func (d *SchedData) SkipDirtyMarkHost() bool {
	return d.IsPublicCloudProvider() || d.IsContainer || d.Hypervisor == SchedTypeContainer
}

func (d *SchedData) IsPublicCloudProvider() bool {
	return PublicCloudProviders.Has(d.Hypervisor)
}

func (d *SchedData) getDiskSize(backend string) int64 {
	total := int64(0)
	for _, disk := range d.Disks {
		if disk.Backend == backend {
			total += disk.Size
		}
	}

	return total
}

func (d *SchedData) AllDiskBackendSize() map[string]int64 {
	backendSizeMap := make(map[string]int64, len(d.Disks))
	for _, disk := range d.Disks {
		newSize := disk.Size
		if size, ok := backendSizeMap[disk.Backend]; ok {
			newSize += size
		}

		backendSizeMap[disk.Backend] = newSize
	}

	return backendSizeMap
}

func parseSize(s string, defaultUnit string, base int) (int64, error) {
	if utils.IsMatch("^\\d+$", s) {
		s = s + defaultUnit
	}
	unit := s[len(s)-1:]
	size, err := strconv.ParseFloat(s[0:len(s)-1], 64)
	if err != nil {
		return 0, err
	}

	if unit == "g" || unit == "G" {
		size *= float64(base * base * base)
	} else if unit == "m" || unit == "M" {
		size *= float64(base * base)
	} else if unit == "k" || unit == "K" {
		size *= float64(base)
	}

	return int64(size), nil
}

func macUnpackHex(mac string) (string, error) {
	d := strings.Split(mac, ":")
	if len(d) < 6 {
		d = strings.Split(mac, "-")
	}
	if len(d) == 6 {
		return strings.Join(d, ""), nil
	}

	return "", fmt.Errorf("Mac unpack hex: format error: %s", mac)
}

func newNetworkFromDesc(desc string) (*Network, error) {
	network := new(Network)
	network.Driver = "virtio"
	network.Idx = ""
	network.Wire = ""
	network.Exit = false
	network.Private = false
	network.Mac = ""
	network.Address = ""
	network.Address6 = ""

	for _, it := range strings.Split(desc, ":") {
		if utils.IsMatchIP4(it) {
			network.Address = it
		} else if utils.IsMatchIP6(it) {
			network.Address6 = it
		} else if utils.IsMatchCompactMacAddr(it) {
			mac, err := macUnpackHex(it)
			if err != nil {
				return nil, err
			}
			network.Mac = mac
		} else if strings.HasPrefix(it, "wire=") {
			network.Wire = it[len("wire="):]
		} else if it == "[random_exit]" {
			network.Exit = true
		} else if it == "[random]" {
			network.Exit = false
		} else if it == "[private]" {
			network.Private = true
		} else if it == "[reserved]" {
			network.Reserved = true
		} else if it == "virtio" || it == "e1000" || it == "sriov" {
			network.Driver = it
		} else if utils.IsMatchSize(it) {
			limit, err := parseSize(it, "M", 1000)
			if err != nil {
				return nil, err
			}
			network.BwLimit = limit / 1000 / 1000
		} else if it == "[vip]" {
			network.Vip = true
		} else {
			network.Idx = it
		}
	}

	return network, nil
}

func (d *SchedData) fillNetworksInfo(sjson *simplejson.Json) error {
	networks := []*Network{}
	for index := 0; ; index++ {
		netIndex := fmt.Sprintf("net.%d", index)
		s, ok := sjson.CheckGet(netIndex)
		if !ok {
			break
		}
		net, err := newNetworkFromDesc(s.MustString())
		if err != nil {
			return err
		}
		networks = append(networks, net)
	}
	d.Networks = networks
	return nil
}

func newIsolatedDeviceFromDesc(desc string) (dev *IsolatedDevice, err error) {
	dev = new(IsolatedDevice)
	for _, it := range strings.Split(desc, ":") {
		if utils.IsMatchUUID(it) {
			dev.ID = it
		} else if ValidPassthroughTypes.Has(it) {
			dev.Type = it
		} else if strings.HasPrefix(it, "vendor=") {
			vendor := it[len("vendor="):]
			vid, ok := IsolatedVendorIDMap[strings.ToUpper(vendor)]
			if !ok {
				vid = vendor
			}
			dev.Vendor = vid
		} else {
			dev.Model = it
		}
	}
	if len(dev.ID) == 0 && len(dev.Model) == 0 {
		dev = nil
		err = fmt.Errorf("Invalid isolated device description: %s", desc)
		return
	}
	return
}

func (d *SchedData) fillIsolatedDeviceInfo(sjson *simplejson.Json) error {
	devs := []*IsolatedDevice{}
	for idx := 0; ; idx++ {
		devIdx := fmt.Sprintf("isolated_device.%d", idx)
		s, ok := sjson.CheckGet(devIdx)
		if !ok {
			break
		}
		dev, err := newIsolatedDeviceFromDesc(s.MustString())
		if err != nil {
			return err
		}
		devs = append(devs, dev)
	}
	d.IsolatedDevices = devs
	return nil
}

func newDiskFromCmdline(dstr string) (*Disk, error) {
	matchFs := func(fs string) bool {
		switch fs {
		case "swap", "ext2", "ext3", "ext4", "xfs", "ntfs", "fat", "hfsplus":
			return true
		default:
			return false
		}
	}

	matchFormat := func(f string) bool {
		switch f {
		case "qcow2", "raw", "docker", "iso", "vmdk":
			return true
		default:
			return false
		}
	}

	matchDriver := func(d string) bool {
		switch d {
		case "virtio", "ide", "scsi", "sata", "pvscsi":
			return true
		default:
			return false
		}
	}

	matchCache := func(c string) bool {
		switch c {
		case "writeback", "none", "writethrough":
			return true
		default:
			return false
		}
	}

	matchBackend := func(b string) bool {
		switch b {
		case "local", "baremetal", "docker":
			return true
		default:
			return false
		}
	}
	disk := &Disk{Backend: "local"}
	for _, nd := range strings.Split(dstr, ":") {
		d := nd
		if utils.IsMatchSize(d) {
			size, err := parseSize(d, "G", 1024)
			if err != nil {
				return nil, err
			}
			disk.Size = size / 1024 / 1024
		}
		if matchFs(d) {
			disk.Fs = &d
		}
		if matchFormat(d) {
			disk.Format = d
		}
		if matchDriver(d) {
			disk.Driver = &d
		}
		if matchCache(d) {
			disk.Cache = &d
		}
		if d == "autoextend" {
			disk.Size = -1
		}
		if matchBackend(d) {
			disk.Backend = d
		}
		if strings.HasPrefix(d, "/") {
			disk.MountPoint = &d
		}
	}
	return disk, nil
}

func newDiskFromSimpleJson(sjson *simplejson.Json, byTest bool) (*Disk, error) {
	disk := new(Disk)
	if byTest {
		return newDiskFromCmdline(sjson.MustString())
	}
	disk.Backend = sjson.Get("backend").MustString()
	disk.ImageID = sjson.Get("image_id").MustString()
	if fs, ok := sjson.CheckGet("fs"); ok {
		if str, err := fs.String(); err == nil {
			disk.Fs = &str
		}
	}
	disk.Os = sjson.Get("os").MustString()
	disk.OSDistribution = sjson.Get("os_distribution").MustString()
	disk.OsVersion = sjson.Get("os_version").MustString()

	disk.Format = sjson.Get("format").MustString()
	if mountPoint, ok := sjson.CheckGet("mountpoint"); ok {
		if str, err := mountPoint.String(); err == nil {
			disk.MountPoint = &str
		}
	}
	if driver, ok := sjson.CheckGet("driver"); ok {
		if str, err := driver.String(); err == nil {
			disk.Driver = &str
		}
	}
	if cache, ok := sjson.CheckGet("cache"); ok {
		if str, err := cache.String(); err == nil {
			disk.Cache = &str
		}
	}
	disk.ImageDiskFormat = sjson.Get("image_disk_format").MustString()
	disk.Size = sjson.Get("size").MustInt64()

	if storage, ok := sjson.CheckGet("storage"); ok {
		if str, err := storage.String(); err == nil {
			disk.Storage = &str
		}
	}
	return disk, nil
}

func newBaremetalDiskConfigFromSimpleJson(sjson *simplejson.Json) (*BaremetalDiskConfig, error) {
	baremetalDiskConfig := new(BaremetalDiskConfig)
	baremetalDiskConfig.Count = sjson.Get("count").MustInt64()
	baremetalDiskConfig.Conf = sjson.Get("conf").MustString()

	rangeArray := make([]int64, 0)
	ranges := sjson.Get("range").MustArray()
	for _, size := range ranges {
		rangeArray = append(rangeArray, size.(int64))
	}
	baremetalDiskConfig.Range = rangeArray

	baremetalDiskConfig.Splits = sjson.Get("splits").MustString()
	baremetalDiskConfig.Strip = sjson.Get("strip").MustInt64()
	baremetalDiskConfig.Type = sjson.Get("type").MustString()
	ada := sjson.Get("adapter").MustInt()
	baremetalDiskConfig.Adapter = &ada
	baremetalDiskConfig.Cachedbadbbu = sjson.Get("cachedbadbbu").MustBool()

	return baremetalDiskConfig, nil
}

func (d *SchedData) fillDisksInfo(sjson *simplejson.Json, byTest bool) error {
	disks := make([]*Disk, 0)
	index := 0
	for {
		diskIndex := fmt.Sprintf("disk.%d", index)
		d, ok := sjson.CheckGet(diskIndex)
		if !ok {
			break
		}
		index++
		disk, err := newDiskFromSimpleJson(d, byTest)
		if err != nil {
			return err
		}
		disks = append(disks, disk)
	}
	if index == 0 {
		return fmt.Errorf("No disk info found in json")
	}
	d.Disks = disks
	return nil
}

func (d *SchedData) fillBaremetalDiskConfig(sjson *simplejson.Json) error {
	baremetalDiskConfigs := []*BaremetalDiskConfig{}
	if d.Hypervisor != HostTypeBaremetal {
		return nil
	}
	config, ok := sjson.CheckGet("baremetal_disk_config")
	if !ok {
		defaultConfs := []*BaremetalDiskConfig{&BaremetalDefaultDiskConfig}
		d.BaremetalDiskConfigs = defaultConfs
		log.V(4).Warningf("No baremetal_disk_config info found in json, use default baremetal disk config: %#v", defaultConfs)
		return nil
	}
	for index := range config.MustArray() {
		baremetalDiskConfig, err := newBaremetalDiskConfigFromSimpleJson(config.GetIndex(index))
		if err != nil {
			return err
		}
		baremetalDiskConfigs = append(baremetalDiskConfigs, baremetalDiskConfig)
	}

	d.BaremetalDiskConfigs = baremetalDiskConfigs
	return nil
}

func (d *SchedData) fillForGuests(sjson *simplejson.Json) {
	newGuest := func(sjson *simplejson.Json) *ForGuest {
		gst := new(ForGuest)
		gst.ID = sjson.Get("id").MustString()
		gst.Name = sjson.Get("name").MustString()
		return gst
	}
	for i := range sjson.Get("for_guests").MustArray() {
		gst := sjson.Get("for_guests").GetIndex(i)
		d.ForGuests = append(d.ForGuests, newGuest(gst))
	}
}

func NewSchedTagFromCmdline(str string) (agg Aggregate, err error) {
	rs := strings.Split(str, ":")
	if len(rs) == 1 || rs[1] == "" {
		err = fmt.Errorf("SchedTag %q no strategy.", str)
		return
	}
	name, strategy := rs[0], rs[1]

	err = AggregateStrategyCheck(strategy)
	if err != nil {
		return
	}

	agg = Aggregate{name, strategy}
	return
}

func (d *SchedData) fillAggregates(sjson *simplejson.Json, byTest bool) error {
	d.Aggregates = []Aggregate{}

	if !byTest {
		if aggNode, ok := sjson.CheckGet("aggregate_strategy"); ok {
			for name, strategy := range aggNode.MustMap() {
				d.Aggregates = append(d.Aggregates, Aggregate{
					Idx: fmt.Sprintf("%v", name), Strategy: fmt.Sprintf("%v", strategy),
				})
			}
		}
	} else {
		index := 0
		for {
			aggIndex := fmt.Sprintf("aggregate.%d", index)
			a, ok := sjson.CheckGet(aggIndex)
			if !ok {
				break
			}
			agg, err := NewSchedTagFromCmdline(a.MustString())
			if err != nil {
				return err
			}
			d.Aggregates = append(d.Aggregates, agg)
			index++
		}
	}

	return nil
}

func (d *SchedData) fillGroupRelations(sjson *simplejson.Json) error {
	d.GroupRelations = []GroupRelation{}

	if aggNode, ok := sjson.CheckGet("group_relations"); ok {
		for index := range aggNode.MustArray() {
			it := aggNode.GetIndex(index)

			r := GroupRelation{}

			if v, ok := it.CheckGet("id"); ok {
				r.GroupID = v.MustString()
			} else {
				return fmt.Errorf("id missing in group relation")
			}

			if v, ok := it.CheckGet("scope"); ok {
				r.Scope = v.MustString()
			} else {
				r.Scope = "host"
			}

			if v, ok := it.CheckGet("strategy"); ok {
				r.Strategy = v.MustString()
			} else {
				r.Strategy = ""
			}

			d.GroupRelations = append(d.GroupRelations, r)
		}
	}

	return nil
}

func (d *SchedData) fillGroups(sjson *simplejson.Json) error {
	d.Groups = []string{}
	if groupsNode, ok := sjson.CheckGet("groups"); ok {
		for index := range groupsNode.MustArray() {
			it := groupsNode.GetIndex(index)
			d.Groups = append(d.Groups, it.MustString())
		}
	}

	return nil
}

// TODO
func (d *SchedData) reviseSchedData() {
	/*if strategy, ok := data.Meta["aggregate_strategy"]; ok {
		d.Aggregates = append(d.Aggregates, Aggregate{
			Idx: fmt.Sprintf("%v", name), Strategy: fmt.Sprintf("%v", strategy),
		})
	}*/
}

type SchedInfo struct {
	Data                  *SchedData      `json:"scheduler"`
	IgnoreFilters         map[string]bool `json:"ignore_filters"`
	SessionID             string          `json:"session_id"`
	IsSuggestion          bool            `json:"suggestion"`
	ShowSuggestionDetails bool            `json:"suggestion_details"`
	SuggestionLimit       int64           `json:"suggestion_limit"`
	SuggestionAll         bool            `json:"suggestion_all"`
	Raw                   string          `json:"raw"`
	BestEffort            bool            `json:"best_effort"`
}

func NewSchedInfo(sjson *simplejson.Json, byTest bool) (*SchedInfo, error) {
	info := new(SchedInfo)

	schedData, ok := sjson.CheckGet("scheduler")
	if !ok {
		return nil, fmt.Errorf("Not found 'scheduler' from request body")

	}

	count := sjson.Get("count").MustInt64(1)

	if data, err := NewSchedData(schedData, count, byTest); err == nil {
		info.Data = data
	} else {
		return nil, err
	}

	info.SessionID = sjson.Get("session_id").MustString()

	ignoreFiltersString := sjson.Get("ignore_filters").MustString()
	filters := strings.Split(ignoreFiltersString, ",")
	ignoreFilters := make(map[string]bool, len(filters))
	for _, filter := range filters {
		ignoreFilters[filter] = true

	}
	info.IgnoreFilters = ignoreFilters

	info.SuggestionLimit = sjson.Get("suggestion_limit").MustInt64()
	if info.SuggestionLimit == 0 {
		info.SuggestionLimit = int64(o.GetOptions().SchedulerTestLimit)
	}
	info.SuggestionAll = sjson.Get("suggestion_all").MustBool()
	info.ShowSuggestionDetails = sjson.Get("suggestion_details").MustBool()
	info.BestEffort = sjson.Get("best_effort").MustBool()

	if bytes, err := sjson.EncodePretty(); err == nil {
		info.Raw = string(bytes)

	}
	return info, nil
}

func (i *SchedInfo) String() (string, error) {
	bytes, err := json.Marshal(i)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

type SchedResultItem interface{}

type SchedResult struct {
	Items []SchedResultItem `json:"scheduler"`
}

type candidateResult struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SchedSuccItem struct {
	Candidate SchedNormalResultItem `json:"candidate"`
}

type SchedErrItem struct {
	Error string `json:"error"`
}

type SchedNormalResultItem struct {
	ID   string                 `json:"id"`
	Name string                 `json:"name"`
	Data map[string]interface{} `json:"data"`
}

type SchedTestResult struct {
	Data   interface{} `json:"data"`
	Total  int64       `json:"total"`
	Limit  int64       `json:"limit"`
	Offset int64       `json:"offset"`
}
