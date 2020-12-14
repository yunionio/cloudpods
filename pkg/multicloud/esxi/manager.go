// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package esxi

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"sync"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
	"golang.org/x/sync/errgroup"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/regutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/multicloud/esxi/vcenter"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

const (
	CLOUD_PROVIDER_VMWARE = api.CLOUD_PROVIDER_VMWARE
)

var (
	defaultDc mo.Datacenter

	defaultDcId = "esxi-default-datacenter"
)

func init() {
	defaultDc.ManagedEntity.Name = "Datacenter"
	defaultDc.ManagedEntity.ExtensibleManagedObject.Self.Type = "Datacenter"
	defaultDc.ManagedEntity.ExtensibleManagedObject.Self.Value = defaultDcId
}

type ESXiClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	host     string
	port     int
	account  string
	password string

	managed bool
}

func NewESXiClientConfig(host string, port int, account, password string) *ESXiClientConfig {
	cfg := &ESXiClientConfig{
		host:     host,
		port:     port,
		account:  account,
		password: password,
	}
	return cfg
}

func (cfg *ESXiClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *ESXiClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *ESXiClientConfig) Managed(managed bool) *ESXiClientConfig {
	cfg.managed = managed
	return cfg
}

type SESXiClient struct {
	*ESXiClientConfig

	cloudprovider.SFakeOnPremiseRegion
	multicloud.SRegion
	multicloud.SNoObjectStorageRegion

	client  *govmomi.Client
	context context.Context

	datacenters     []*SDatacenter
	networkQueryMap *sync.Map
}

func NewESXiClient(cfg *ESXiClientConfig) (*SESXiClient, error) {
	cfg.Managed(true)
	return NewESXiClient2(cfg)
}

func NewESXiClient2(cfg *ESXiClientConfig) (*SESXiClient, error) {
	cli := &SESXiClient{
		ESXiClientConfig: cfg,
		context:          context.Background(),
	}

	err := cli.connect()
	if err != nil {
		return nil, err
	}

	if !cli.IsVCenter() {
		err := cli.checkHostManagedByVCenter()
		if err != nil {
			if cfg.managed {
				cli.disconnect()
				return nil, err
			} else {
				log.Warningf("%s", err)
			}
		}
	}
	return cli, nil
}

func NewESXiClientFromJson(ctx context.Context, input jsonutils.JSONObject) (*SESXiClient, *vcenter.SVCenterAccessInfo, error) {
	accessInfo := new(vcenter.SVCenterAccessInfo)
	err := input.Unmarshal(accessInfo)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "unmarshal SVCenterAccessInfo: %s", input)
	}
	c, err := NewESXiClientFromAccessInfo(ctx, accessInfo)
	return c, accessInfo, err
}

func NewESXiClientFromAccessInfo(ctx context.Context, accessInfo *vcenter.SVCenterAccessInfo) (*SESXiClient, error) {
	if len(accessInfo.VcenterId) > 0 {
		tmp, err := utils.DescryptAESBase64(accessInfo.VcenterId, accessInfo.Password)
		if err == nil {
			accessInfo.Password = tmp
		}
	}
	client, err := NewESXiClient(
		NewESXiClientConfig(
			accessInfo.Host,
			accessInfo.Port,
			accessInfo.Account,
			accessInfo.Password,
		).Managed(true),
	)
	if err != nil {
		return nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	client.context = ctx
	return client, nil
}

func (cli *SESXiClient) getUrl() string {
	if cli.port == 443 || cli.port == 0 {
		return fmt.Sprintf("https://%s", cli.host)
	} else {
		return fmt.Sprintf("https://%s:%d", cli.host, cli.port)
	}
}

func (cli *SESXiClient) url() string {
	return fmt.Sprintf("%s/sdk", cli.getUrl())
}

func (cli *SESXiClient) connect() error {
	u, err := url.Parse(cli.url())
	if err != nil {
		return fmt.Errorf("Illegal url %s: %s", cli.url(), err)
	}

	var govmcli *govmomi.Client
	{
		insecure := true
		soapCli := soap.NewClient(u, insecure)
		httpClient := &soapCli.Client
		httputils.SetClientProxyFunc(httpClient, cli.cpcfg.ProxyFunc)
		vimCli, err := vim25.NewClient(cli.context, soapCli)
		if err != nil {
			return err
		}
		govmcli = &govmomi.Client{
			Client:         vimCli,
			SessionManager: session.NewManager(vimCli),
		}
	}

	userinfo := url.UserPassword(cli.account, cli.password)

	err = govmcli.Login(cli.context, userinfo)
	if err != nil {
		return err
	}

	cli.client = govmcli

	defaultDc.ManagedEntity.Parent = &cli.client.ServiceContent.RootFolder

	return nil
}

func (cli *SESXiClient) disconnect() error {
	if cli.client != nil {
		return cli.client.Logout(cli.context)
	}
	return nil
}

func (cli *SESXiClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	err := cli.connect()
	if err != nil {
		return nil, err
	}
	subAccount := cloudprovider.SSubAccount{
		Account:      cli.account,
		Name:         cli.cpcfg.Name,
		HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (cli *SESXiClient) GetAccountId() string {
	return fmt.Sprintf("%s@%s:%d", cli.account, cli.host, cli.port)
}

func (cli *SESXiClient) GetVersion() string {
	return cli.client.ServiceContent.About.Version
}

func (cli *SESXiClient) About() jsonutils.JSONObject {
	about := jsonutils.Marshal(&cli.client.ServiceContent.About)
	aboutDict := about.(*jsonutils.JSONDict)
	aboutDict.Add(jsonutils.NewString(cli.getEndpointType()), "endpoint_type")
	return aboutDict
}

func (cli *SESXiClient) getEndpointType() string {
	if cli.IsVCenter() {
		return "VCenter"
	} else {
		return "ESXi"
	}
}

func (cli *SESXiClient) GetUUID() string {
	about := cli.client.ServiceContent.About
	return about.InstanceUuid
}

func (cli *SESXiClient) fetchDatacentersFromHosts() error {
	var hosts []mo.HostSystem
	err := cli.scanAllMObjects(HOST_SYSTEM_PROPS, &hosts)
	if err != nil {
		return errors.Wrap(err, "cli.scanAllMObjects host")
	}
	dcList := make([]*SDatacenter, 0)
	for i := 0; i < len(hosts); i += 1 {
		me := newManagedObject(cli, &hosts[i], nil)
		dcme := me.findInParents("Datacenter")
		if dcme == nil {
			return cloudprovider.ErrNotFound
		}
		_, err := findDatacenterByMoId(dcList, dcme.Self.Value)
		if err == nil {
			continue
		}
		var moDc mo.Datacenter
		err = cli.reference2Object(dcme.Self, DATACENTER_PROPS, &moDc)
		if err != nil {
			return errors.Wrap(err, "cli.reference2Object")
		}
		dc, err := cli.newDatacenterFromMo(&moDc)
		if err != nil {
			return errors.Wrap(err, "cli.newDatacenterFromMo")
		}
		dcList = append(dcList, dc)
	}
	cli.datacenters = dcList
	return nil
}

func (cli *SESXiClient) fetchFakeDatacenter() error {
	dc, err := cli.newDatacenterFromMo(&defaultDc)
	if err != nil {
		return errors.Wrap(err, "newDatacenterFromMo")
	}
	cli.datacenters = []*SDatacenter{dc}
	return nil
}

func (cli *SESXiClient) fetchDatacenters() error {
	var dcs []mo.Datacenter
	err := cli.scanAllMObjects(DATACENTER_PROPS, &dcs)
	if err != nil {
		// log.Debugf("cli.scanAllMObjects datacenter fail %s, try cli.fetchDatacentersFromHosts", err)
		// err := cli.fetchDatacentersFromHosts()
		// if err != nil {
		log.Debugf("cli.fetchDatacentersFromHosts fail %s, try cli.fetchFakeDatacenter", err)
		return cli.fetchFakeDatacenter()
		// }
	}
	cli.datacenters = make([]*SDatacenter, len(dcs))
	for i := 0; i < len(dcs); i += 1 {
		dc, err := cli.newDatacenterFromMo(&dcs[i])
		if err != nil {
			return errors.Wrap(err, "cli.newDatacenterFromMo")
		}
		cli.datacenters[i] = dc
	}
	return nil
}

func (cli *SESXiClient) newDatacenterFromMo(mo *mo.Datacenter) (*SDatacenter, error) {
	dc := newDatacenter(cli, mo)
	err := dc.scanHosts()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanHosts")
	}
	err = dc.scanDatastores()
	if err != nil {
		return nil, errors.Wrap(err, "dc.scanDatastores")
	}
	return dc, nil
}

func (cli *SESXiClient) scanAllMObjects(props []string, dst interface{}) error {
	return cli.scanMObjects(cli.client.ServiceContent.RootFolder, props, dst)
}

func (cli *SESXiClient) SearchVM(id string) (*SVirtualMachine, error) {
	filter := property.Filter{}
	filter["summary.config.uuid"] = id
	var movms []mo.VirtualMachine
	err := cli.scanMObjectsWithFilter(cli.client.ServiceContent.RootFolder, VIRTUAL_MACHINE_PROPS, &movms, filter)
	if err != nil {
		return nil, err
	}
	if len(movms) == 0 {
		return nil, errors.ErrNotFound
	}
	vm := NewVirtualMachine(cli, &movms[0], nil)
	dc, err := vm.fetchDatacenter()
	if err != nil {
		return nil, errors.Wrap(err, "fetchDatacenter")
	}
	vm.datacenter = dc
	return vm, nil
}

func (cli *SESXiClient) SearchTemplateVM(id string) (*SVirtualMachine, error) {
	filter := property.Filter{}
	uuid := toTemplateUuid(id)
	filter["summary.config.uuid"] = uuid
	var movms []mo.VirtualMachine
	err := cli.scanMObjectsWithFilter(cli.client.ServiceContent.RootFolder, VIRTUAL_MACHINE_PROPS, &movms, filter)
	if err != nil {
		return nil, err
	}
	if len(movms) == 0 {
		return nil, errors.ErrNotFound
	}
	vm := NewVirtualMachine(cli, &movms[0], nil)
	if !vm.IsTemplate() {
		return nil, errors.ErrNotFound
	}
	dc, err := vm.fetchDatacenter()
	if err != nil {
		return nil, errors.Wrap(err, "fetchDatacenter")
	}
	vm.datacenter = dc
	return vm, nil
}

func (cli *SESXiClient) scanMObjectsWithFilter(folder types.ManagedObjectReference, props []string, dst interface{}, filter property.Filter) error {
	dstValue := reflect.Indirect(reflect.ValueOf(dst))
	dstType := dstValue.Type()
	dstEleType := dstType.Elem()

	resType := dstEleType.Name()
	m := view.NewManager(cli.client.Client)

	v, err := m.CreateContainerView(cli.context, folder, []string{resType}, true)
	if err != nil {
		return errors.Wrapf(err, "m.CreateContainerView %s", resType)
	}

	defer v.Destroy(cli.context)

	err = v.RetrieveWithFilter(cli.context, []string{resType}, props, dst, filter)
	if err != nil {
		// hack
		if strings.Contains(err.Error(), "object references is empty") {
			return nil
		}
		return errors.Wrapf(err, "v.RetrieveWithFilter %s", resType)
	}

	return nil
}

func (cli *SESXiClient) scanMObjects(folder types.ManagedObjectReference, props []string, dst interface{}) error {
	dstValue := reflect.Indirect(reflect.ValueOf(dst))
	dstType := dstValue.Type()
	dstEleType := dstType.Elem()

	resType := dstEleType.Name()
	m := view.NewManager(cli.client.Client)

	v, err := m.CreateContainerView(cli.context, folder, []string{resType}, true)
	if err != nil {
		return errors.Wrapf(err, "m.CreateContainerView %s", resType)
	}

	defer v.Destroy(cli.context)

	err = v.Retrieve(cli.context, []string{resType}, props, dst)
	if err != nil {
		return errors.Wrapf(err, "v.Retrieve %s", resType)
	}

	return nil
}

func (cli *SESXiClient) references2Objects(refs []types.ManagedObjectReference, props []string, dst interface{}) error {
	pc := property.DefaultCollector(cli.client.Client)
	err := pc.Retrieve(cli.context, refs, props, dst)
	if err != nil {
		log.Errorf("pc.Retrieve fail %s", err)
		return err
	}
	return nil
}

func (cli *SESXiClient) reference2Object(ref types.ManagedObjectReference, props []string, dst interface{}) error {
	pc := property.DefaultCollector(cli.client.Client)
	err := pc.RetrieveOne(cli.context, ref, props, dst)
	if err != nil {
		return errors.Wrap(err, "pc.RetrieveOne")
	}
	return nil
}

func (cli *SESXiClient) GetDatacenters() ([]*SDatacenter, error) {
	if cli.datacenters == nil {
		err := cli.fetchDatacenters()
		if err != nil {
			return nil, err
		}
	}
	return cli.datacenters, nil
}

func (cli *SESXiClient) FindDatacenterByMoId(dcId string) (*SDatacenter, error) {
	dcs, err := cli.GetDatacenters()
	if err != nil {
		return nil, err
	}
	return findDatacenterByMoId(dcs, dcId)
}

func findDatacenterByMoId(dcs []*SDatacenter, dcId string) (*SDatacenter, error) {
	for i := 0; i < len(dcs); i += 1 {
		if dcs[i].GetId() == dcId {
			return dcs[i], nil
		}
		// defaultDcId means no premision to get datacenter, so return fake dc
		if dcs[i].GetId() == defaultDcId {
			return dcs[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SESXiClient) GetIProjects() ([]cloudprovider.ICloudProject, error) {
	dcs, err := cli.GetDatacenters()
	if err != nil {
		return nil, errors.Wrap(err, "GetDatacenters")
	}
	ret := []cloudprovider.ICloudProject{}
	for i := 0; i < len(dcs); i++ {
		iprojects, err := dcs[i].GetResourcePools()
		if err != nil {
			return nil, errors.Wrap(err, "GetResourcePools")
		}
		ret = append(ret, iprojects...)
	}
	return ret, nil
}

func (cli *SESXiClient) FindHostByMoId(moId string) (cloudprovider.ICloudHost, error) {
	dcs, err := cli.GetDatacenters()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(dcs); i += 1 {
		ihost, err := dcs[i].GetIHostByMoId(moId)
		if err == nil {
			return ihost, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SESXiClient) getPrivateId(idStr string) string {
	if len(cli.cpcfg.Id) > 0 && strings.HasPrefix(idStr, cli.cpcfg.Id) {
		idStr = idStr[len(cli.cpcfg.Id)+1:]
	}
	return idStr
}

func (cli *SESXiClient) checkHostManagedByVCenter() error {
	host, err := cli.FindHostByIp(cli.host)
	if err != nil {
		if errors.Cause(err) == errors.ErrNotFound {
			// host might be behind a NAT
			return nil
		}
		return errors.Wrap(err, "cli.FindHostByIp")
	}
	if host.IsManagedByVCenter() {
		return fmt.Errorf("ESXi host is managed by vcenter %s, please connect to vcenter instead for full management functions!", host.GetManagementServerIp())
	}
	return nil
}

func (cli *SESXiClient) IsHostIpExists(hostIp string) (bool, error) {
	searchIndex := object.NewSearchIndex(cli.client.Client)

	hostRef, err := searchIndex.FindByIp(cli.context, nil, cli.getPrivateId(hostIp), false)
	if err != nil {
		log.Errorf("searchIndex.FindByIp fail %s", err)
		return false, err
	}
	if hostRef == nil {
		return false, nil
	}
	return true, nil
}

func (cli *SESXiClient) FindHostByIp(hostIp string) (*SHost, error) {
	searchIndex := object.NewSearchIndex(cli.client.Client)

	hostRef, err := searchIndex.FindByIp(cli.context, nil, cli.getPrivateId(hostIp), false)
	if err != nil {
		log.Errorf("searchIndex.FindByIp fail %s", err)
		return nil, errors.Wrap(err, "searchIndex.FindByIp")
	}

	if hostRef == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "cannot find %s", cli.getPrivateId(hostIp))
	}

	var host mo.HostSystem
	err = cli.reference2Object(hostRef.Reference(), HOST_SYSTEM_PROPS, &host)
	if err != nil {
		log.Errorf("reference2Object fail %s", err)
		return nil, errors.Wrap(err, "cli.reference2Object")
	}

	h := NewHost(cli, &host, nil)
	if h == nil {
		return nil, errors.Wrap(errors.ErrInvalidStatus, "empty host mo")
	}
	return h, nil
}

func (cli *SESXiClient) acquireCloneTicket() (string, error) {
	manager := session.NewManager(cli.client.Client)
	return manager.AcquireCloneTicket(cli.context)
}

func (cli *SESXiClient) IsVCenter() bool {
	return cli.client.Client.IsVC()
}

func (cli *SESXiClient) IsValid() bool {
	return cli.client.Client.Valid()
}

func (cli *SESXiClient) GetCapabilities() []string {
	caps := []string{
		cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		// cloudprovider.CLOUD_CAPABILITY_NETWORK,
		// cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		// cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
		// cloudprovider.CLOUD_CAPABILITY_EVENT,
	}
	return caps
}

func (cli *SESXiClient) FindVMByPrivateID(idstr string) (*SVirtualMachine, error) {
	searchIndex := object.NewSearchIndex(cli.client.Client)
	instanceUuid := true
	vmRef, err := searchIndex.FindByUuid(cli.context, nil, idstr, true, &instanceUuid)
	if err != nil {
		return nil, errors.Wrap(err, "searchIndex.FindByUuid fail")
	}
	if vmRef == nil {
		return nil, fmt.Errorf("cannot find %s", idstr)
	}
	var vm mo.VirtualMachine
	err = cli.reference2Object(vmRef.Reference(), VIRTUAL_MACHINE_PROPS, &vm)
	if err != nil {
		return nil, errors.Wrap(err, "reference2Object fail")
	}

	ret := NewVirtualMachine(cli, &vm, nil)
	if ret == nil {
		return nil, errors.Error("invalid vm")
	}
	return ret, nil
}

func (cli *SESXiClient) DoExtendDiskOnline(_vm *SVirtualMachine, _disk *SVirtualDisk, newSizeMb int64) error {
	disk := _disk.getVirtualDisk()
	disk.CapacityInKB = newSizeMb * 1024
	devSepc := types.VirtualDeviceConfigSpec{Operation: types.VirtualDeviceConfigSpecOperationEdit, Device: disk}
	spec := types.VirtualMachineConfigSpec{DeviceChange: []types.BaseVirtualDeviceConfigSpec{&devSepc}}
	vm := object.NewVirtualMachine(cli.client.Client, _vm.getVirtualMachine().Reference())
	task, err := vm.Reconfigure(cli.context, spec)
	if err != nil {
		return errors.Wrapf(err, "vm reconfigure failed")
	}
	return task.Wait(cli.context)
}

func (cli *SESXiClient) ExtendDisk(url string, newSizeMb int64) error {
	param := types.ExtendVirtualDisk_Task{
		This:          *cli.client.Client.ServiceContent.VirtualDiskManager,
		Name:          url,
		NewCapacityKb: newSizeMb * 1024,
	}
	response, err := methods.ExtendVirtualDisk_Task(cli.context, cli.client, &param)
	if err != nil {
		return errors.Wrapf(err, "extend virtualdisk task failed")
	}
	log.Debugf("extend virtual disk task response: %s", response.Returnval.String())
	return nil
}

func (cli *SESXiClient) CopyDisk(ctx context.Context, src, dst string, isForce bool) error {
	dm := object.NewVirtualDiskManager(cli.client.Client)
	task, err := dm.CopyVirtualDisk(ctx, src, nil, dst, nil, nil, isForce)
	if err != nil {
		return errors.Wrap(err, "CopyVirtualDisk")
	}
	return task.Wait(ctx)
}

func (cli *SESXiClient) MoveDisk(ctx context.Context, src, dst string, isForce bool) error {
	dm := object.NewVirtualDiskManager(cli.client.Client)
	task, err := dm.MoveVirtualDisk(ctx, src, nil, dst, nil, isForce)
	if err != nil {
		return errors.Wrap(err, "MoveVirtualDisk")
	}
	return task.Wait(ctx)
}

var (
	SIMPLE_HOST_PROPS    = []string{"name", "config.network", "vm"}
	SIMPLE_VM_PROPS      = []string{"name", "guest.net", "config.template", "config.hardware.device"}
	SIMPLE_DVPG_PROPS    = []string{"key", "config.defaultPortConfig", "config.distributedVirtualSwitch"}
	SIMPLE_NETWORK_PROPS = []string{"name"}
)

type SIPVlan struct {
	IP     netutils.IPV4Addr
	VlanId int32
}

type SSimpleVM struct {
	Name    string
	IPVlans []SIPVlan
}

func (cli *SESXiClient) scanAllDvPortgroups() ([]*SDistributedVirtualPortgroup, error) {
	var modvpgs []mo.DistributedVirtualPortgroup
	err := cli.scanAllMObjects(SIMPLE_DVPG_PROPS, &modvpgs)
	if err != nil {
		return nil, errors.Wrap(err, "scanAllMObjects")
	}
	dvpgs := make([]*SDistributedVirtualPortgroup, 0, len(modvpgs))
	for i := range modvpgs {
		dvpgs = append(dvpgs, NewDistributedVirtualPortgroup(cli, &modvpgs[i], nil))
	}
	return dvpgs, nil
}

func (cli *SESXiClient) scanAllNetworks() ([]mo.Network, error) {
	var monets []mo.Network
	err := cli.scanAllMObjects(SIMPLE_NETWORK_PROPS, &monets)
	if err != nil {
		return nil, errors.Wrap(err, "scanAllMObjects")
	}
	return monets, nil
}

func (cli *SESXiClient) networkName(refV string) (string, error) {
	if cli.networkQueryMap == nil {
		nets, err := cli.scanAllNetworks()
		if err != nil {
			return "", err
		}
		cli.networkQueryMap = &sync.Map{}
		for i := range nets {
			cli.networkQueryMap.Store(nets[i].Reference().Value, nets[i].Name)
		}
	}
	iter, ok := cli.networkQueryMap.Load(refV)
	if !ok {
		return "", nil
	}
	return iter.(string), nil
}

func (cli *SESXiClient) HostVmIPsInDc(ctx context.Context, dc *SDatacenter) (SNetworkInfo, error) {
	var hosts []mo.HostSystem
	err := cli.scanMObjects(dc.object.Entity().Self, HOST_SYSTEM_PROPS, &hosts)
	if err != nil {
		return SNetworkInfo{}, errors.Wrap(err, "scanMObjects")
	}
	return cli.hostVMIPs(ctx, hosts)
}

func (cli *SESXiClient) HostVmIPsInCluster(ctx context.Context, cluster *SCluster) (SNetworkInfo, error) {
	var hosts []mo.HostSystem
	err := cli.scanMObjects(cluster.object.Entity().Self, HOST_SYSTEM_PROPS, &hosts)
	if err != nil {
		return SNetworkInfo{}, errors.Wrap(err, "scanMObjects")
	}
	return cli.hostVMIPs(ctx, hosts)
}

type SNetworkInfo struct {
	HostIps map[string]netutils.IPV4Addr
	VMs     []SSimpleVM
	VlanIps map[int32][]netutils.IPV4Addr
	IPPool  SIPPool
}

func (cli *SESXiClient) hostVMIPs(ctx context.Context, hosts []mo.HostSystem) (SNetworkInfo, error) {
	ret := SNetworkInfo{}
	dvpgMap, err := cli.getDVPGMap()
	if err != nil {
		return ret, errors.Wrap(err, "unable to get dvpgKeyVlanMap")
	}
	group, ctx := errgroup.WithContext(ctx)
	collection := make([]SNetworkInfo, len(hosts))
	for i := range hosts {
		j := i
		group.Go(func() error {
			nInfo, err := cli.vmIPs(&hosts[j], dvpgMap)
			if err != nil {
				return err
			}
			collection[j] = nInfo
			return nil
		})
	}

	hostIps := make(map[string]netutils.IPV4Addr, len(hosts))
	for i := range hosts {
		// find ip
		host := &SHost{SManagedObject: newManagedObject(cli, &hosts[i], nil)}
		ipStr := host.GetAccessIp()
		ip, err := netutils.NewIPV4Addr(ipStr)
		if err != nil {
			return ret, errors.Wrapf(err, "invalid host ip %q", ipStr)
		}
		hostIps[host.GetName()] = ip
	}
	err = group.Wait()
	if err != nil {
		return ret, err
	}
	// length
	if len(collection) == 0 {
		ret.HostIps = hostIps
		return ret, nil
	}
	ni := cli.mergeNetworInfo(collection)
	ni.HostIps = hostIps
	return ni, nil
}

func (cli *SESXiClient) mergeNetworInfo(nInfos []SNetworkInfo) SNetworkInfo {
	var vmsLen, vlanIpLen, ipPoolLen int
	for i := range nInfos {
		vmsLen += len(nInfos[i].VMs)
		vlanIpLen += len(nInfos[i].VlanIps)
		ipPoolLen += nInfos[i].IPPool.Len()
	}
	ret := SNetworkInfo{
		VMs:     make([]SSimpleVM, 0, vmsLen),
		VlanIps: make(map[int32][]netutils.IPV4Addr, vlanIpLen),
		IPPool:  NewIPPool(ipPoolLen),
	}
	for i := range nInfos {
		ret.VMs = append(ret.VMs, nInfos[i].VMs...)
		for vlan, ips := range nInfos[i].VlanIps {
			ret.VlanIps[vlan] = append(ret.VlanIps[vlan], ips...)
		}
		ret.IPPool.Merge(&nInfos[i].IPPool)
	}
	for _, ips := range ret.VlanIps {
		sort.Slice(ips, func(i, j int) bool {
			return ips[i] < ips[j]
		})
	}
	return ret
}

func (cli *SESXiClient) HostVmIPs(ctx context.Context) (SNetworkInfo, error) {
	var hosts []mo.HostSystem
	err := cli.scanAllMObjects(SIMPLE_HOST_PROPS, &hosts)
	if err != nil {
		return SNetworkInfo{}, errors.Wrap(err, "scanAllMObjects")
	}
	return cli.hostVMIPs(ctx, hosts)
}

func (cli *SESXiClient) macVlanMap(mohost *mo.HostSystem, movm *mo.VirtualMachine, dvpgMap sVPGMap) map[string]int32 {
	vpgMap := cli.getVPGMap(mohost)
	ret := make(map[string]int32, 2)
	for _, device := range movm.Config.Hardware.Device {
		bcard, ok := device.(types.BaseVirtualEthernetCard)
		if !ok {
			continue
		}
		card := bcard.GetVirtualEthernetCard()
		mac := card.MacAddress
		switch bk := card.Backing.(type) {
		case *types.VirtualEthernetCardDistributedVirtualPortBackingInfo:
			key := bk.Port.PortgroupKey
			proc, ok := dvpgMap.Get(key)
			if !ok {
				log.Errorf("dvpg %s not found in key-vlanid map", key)
				ret[mac] = 0
				continue
			}
			ret[mac] = proc.vlanId
		case *types.VirtualEthernetCardNetworkBackingInfo:
			netName, err := cli.networkName(bk.Network.Value)
			if err != nil {
				log.Errorf("get getNetworkName of %q: %v", bk.Network.Value, err)
				ret[mac] = 0
				continue
			}
			key := fmt.Sprintf("%s-%s", mohost.Reference().Value, netName)
			proc, ok := vpgMap.Get(key)
			if !ok {
				log.Errorf("vpg %s not found in key-vlanid map", key)
				ret[mac] = 0
				continue
			}
			ret[mac] = proc.vlanId
		default:
			ret[mac] = 0
		}
	}
	return ret
}

func (cli *SESXiClient) vmIPs(host *mo.HostSystem, vpgMap sVPGMap) (SNetworkInfo, error) {
	nInfo := SNetworkInfo{
		VMs:     make([]SSimpleVM, 0, len(host.Vm)),
		VlanIps: make(map[int32][]netutils.IPV4Addr),
		IPPool:  NewIPPool(),
	}
	if len(host.Vm) == 0 {
		return nInfo, nil
	}
	var vms []mo.VirtualMachine
	err := cli.references2Objects(host.Vm, SIMPLE_VM_PROPS, &vms)
	if err != nil {
		return nInfo, errors.Wrap(err, "references2Objects")
	}
	for i := range vms {
		vm := vms[i]
		if vm.Config == nil || vm.Config.Template {
			continue
		}
		if vm.Guest == nil {
			continue
		}
		macVlanMap := cli.macVlanMap(host, &vm, vpgMap)
		guestIps := make([]SIPVlan, 0)
		for _, net := range vm.Guest.Net {
			if len(net.Network) == 0 {
				continue
			}
			mac := net.MacAddress
			for _, ip := range net.IpAddress {
				if !regutils.MatchIP4Addr(ip) {
					continue
				}
				if !vmIPV4Filter.Contains(ip) {
					continue
				}
				ipaddr, _ := netutils.NewIPV4Addr(ip)
				if netutils.IsLinkLocal(ipaddr) {
					continue
				}
				vlan := macVlanMap[mac]
				guestIps = append(guestIps, SIPVlan{
					IP:     ipaddr,
					VlanId: vlan,
				})
				nInfo.VlanIps[vlan] = append(nInfo.VlanIps[vlan], ipaddr)
				nInfo.IPPool.Insert(ipaddr, SIPProc{
					VlanId: vlan,
				})
				break
			}
		}
		nInfo.VMs = append(nInfo.VMs, SSimpleVM{vm.Name, guestIps})
	}
	return nInfo, nil
}
