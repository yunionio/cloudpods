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
	"strings"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/session"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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

type SESXiClient struct {
	cloudprovider.SFakeOnPremiseRegion
	multicloud.SRegion
	multicloud.SNoObjectStorageRegion

	providerId   string
	providerName string
	host         string
	port         int
	account      string
	password     string
	client       *govmomi.Client
	context      context.Context

	datacenters []*SDatacenter
}

func NewESXiClient(providerId string, providerName string, host string, port int, account string, passwd string) (*SESXiClient, error) {
	return NewESXiClient2(providerId, providerName, host, port, account, passwd, true)
}

func NewESXiClient2(providerId string, providerName string, host string, port int, account string, passwd string, managed bool) (*SESXiClient, error) {
	cli := &SESXiClient{providerId: providerId, providerName: providerName,
		host: host, port: port, account: account, password: passwd, context: context.Background()}

	err := cli.connect()
	if err != nil {
		return nil, err
	}

	if !cli.IsVCenter() {
		err := cli.checkHostManagedByVCenter()
		if err != nil {
			if managed {
				cli.disconnect()
				return nil, err
			} else {
				log.Warningf("%s", err)
			}
		}
	}
	return cli, nil
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

	govmcli, err := govmomi.NewClient(cli.context, u, true)
	if err != nil {
		return err
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
		Name:         cli.providerName,
		HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (cli *SESXiClient) GetAccountId() string {
	return cli.account
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
	if len(cli.providerId) > 0 && strings.HasPrefix(idStr, cli.providerId) {
		idStr = idStr[len(cli.providerId)+1:]
	}
	return idStr
}

func (cli *SESXiClient) checkHostManagedByVCenter() error {
	host, err := cli.FindHostByIp(cli.host)
	if err != nil {
		return err
	}
	if host.IsManagedByVCenter() {
		return fmt.Errorf("ESXi host is managed by vcenter %s, please connect to vcenter instead for full management functions!", host.GetManagementServerIp())
	}
	return nil
}

func (cli *SESXiClient) FindHostByIp(hostIp string) (*SHost, error) {
	searchIndex := object.NewSearchIndex(cli.client.Client)

	hostRef, err := searchIndex.FindByIp(cli.context, nil, cli.getPrivateId(hostIp), false)
	if err != nil {
		log.Errorf("searchIndex.FindByIp fail %s", err)
		return nil, err
	}

	if hostRef == nil {
		return nil, fmt.Errorf("cannot find %s", cli.getPrivateId(hostIp))
	}

	var host mo.HostSystem
	err = cli.reference2Object(hostRef.Reference(), HOST_SYSTEM_PROPS, &host)
	if err != nil {
		log.Errorf("reference2Object fail %s", err)
		return nil, err
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
		// cloudprovider.CLOUD_CAPABILITY_PROJECT,
		cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		// cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		// cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
	}
	return caps
}
