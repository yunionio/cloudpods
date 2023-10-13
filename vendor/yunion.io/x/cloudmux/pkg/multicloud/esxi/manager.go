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
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"
	"strings"
	"time"

	xj "github.com/basgys/goxml2json"
	"github.com/fatih/color"
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
	"moul.io/http2curl/v2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/cloudmux/pkg/multicloud/esxi/vcenter"
)

const (
	CLOUD_PROVIDER_VMWARE = api.CLOUD_PROVIDER_VMWARE
)

var (
	defaultDc mo.Datacenter

	defaultDcId = "esxi-default-datacenter"
)

var _ cloudprovider.ICloudRegion = (*SESXiClient)(nil)

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
	debug   bool
	format  string
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

func (cfg *ESXiClientConfig) Debug(debug bool) *ESXiClientConfig {
	cfg.debug = debug
	return cfg
}

func (cfg *ESXiClientConfig) Format(format string) *ESXiClientConfig {
	cfg.format = format
	return cfg
}

func (cfg *ESXiClientConfig) Managed(managed bool) *ESXiClientConfig {
	cfg.managed = managed
	return cfg
}

type SESXiClient struct {
	*ESXiClientConfig

	multicloud.SRegion
	multicloud.SRegionEipBase
	multicloud.SNoObjectStorageRegion
	multicloud.SRegionLbBase

	client  *govmomi.Client
	context context.Context

	datacenters []*SDatacenter

	fakeVpc *sFakeVpc
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

	cli.fakeVpc = &sFakeVpc{
		client: cli,
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

var (
	red    = color.New(color.FgRed, color.Bold).PrintlnFunc()
	green  = color.New(color.FgGreen, color.Bold).PrintlnFunc()
	yellow = color.New(color.FgYellow, color.Bold).PrintlnFunc()
	cyan   = color.New(color.FgHiCyan, color.Bold).PrintlnFunc()
)

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
		transport := &http.Transport{
			// 代理设置
			Proxy: cli.cpcfg.ProxyFunc,

			ExpectContinueTimeout: 5 * time.Second,
			TLSClientConfig:       &tls.Config{InsecureSkipVerify: insecure},
			IdleConnTimeout:       time.Minute * 1,
			TLSHandshakeTimeout:   time.Minute * 1,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		}
		httpClient.Transport = cloudprovider.GetCheckTransport(transport, func(req *http.Request) (func(resp *http.Response) error, error) {
			if cli.debug {
				dump, _ := httputil.DumpRequestOut(req, false)
				yellow(string(dump))
				// 忽略掉上传文件的请求,避免大量日志输出
				if req.Header.Get("Content-Type") != "application/octet-stream" {
					curlCmd, _ := http2curl.GetCurlCommand(req)
					cyan("CURL:", curlCmd, "\n")
				}
			}
			respCheck := func(resp *http.Response) error {
				if cli.debug {
					dump, _ := httputil.DumpResponse(resp, true)
					body := string(dump)
					if cli.format == "json" {
						obj, err := xj.Convert(bytes.NewReader(dump))
						if err == nil {
							body = string(obj.String())
						}
					}
					if resp.StatusCode < 300 {
						green(body)
					} else if resp.StatusCode < 400 {
						yellow(body)
					} else {
						red(body)
					}
				}
				return nil
			}
			return respCheck, nil
		})
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

func (cli *SESXiClient) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(cli.GetName()).CN(cli.GetName())
	return table
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
	// err = dc.scanDatastores()
	// if err != nil {
	// 	return nil, errors.Wrap(err, "dc.scanDatastores")
	// }
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

func fixProps(props []string) []string {
	for _, key := range []string{
		"name", "parent",
	} {
		if !utils.IsInArray(key, props) {
			props = append(props, key)
		}
	}
	return props
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

	err = v.Retrieve(cli.context, []string{resType}, fixProps(props), dst)
	if err != nil {
		return errors.Wrapf(err, "v.Retrieve %s", resType)
	}

	return nil
}

func (cli *SESXiClient) references2Objects(refs []types.ManagedObjectReference, props []string, dst interface{}) error {
	pc := property.DefaultCollector(cli.client.Client)
	err := pc.Retrieve(cli.context, refs, fixProps(props), dst)
	if err != nil {
		return errors.Wrapf(err, "Retrieve")
	}
	return nil
}

func (cli *SESXiClient) reference2Object(ref types.ManagedObjectReference, props []string, dst interface{}) error {
	pc := property.DefaultCollector(cli.client.Client)
	err := pc.RetrieveOne(cli.context, ref, fixProps(props), dst)
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
		return false, errors.Wrapf(err, "FindByIp %s", hostIp)
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
		return nil, errors.Wrapf(err, "FindByIp %s", hostIp)
	}

	if hostRef == nil {
		return nil, errors.Wrapf(errors.ErrNotFound, "cannot find %s", cli.getPrivateId(hostIp))
	}

	var host mo.HostSystem
	err = cli.reference2Object(hostRef.Reference(), HOST_SYSTEM_PROPS, &host)
	if err != nil {
		return nil, errors.Wrapf(err, "reference2Object %s", hostIp)
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
		cloudprovider.CLOUD_CAPABILITY_NETWORK,
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
