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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	CLOUD_PROVIDER_VMWARE = api.CLOUD_PROVIDER_VMWARE
)

type SESXiClient struct {
	cloudprovider.SFakeOnPremiseRegion

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
	cli := &SESXiClient{providerId: providerId, providerName: providerName,
		host: host, port: port, account: account, password: passwd, context: context.Background()}

	err := cli.connect()
	if err != nil {
		return nil, err
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

func (cli *SESXiClient) fetchDatacenters() error {
	var dcs []mo.Datacenter
	err := cli.scanAllMObjects(DATACENTER_PROPS, &dcs)
	if err != nil {
		return err
	}
	cli.datacenters = make([]*SDatacenter, len(dcs))
	for i := 0; i < len(dcs); i += 1 {
		cli.datacenters[i] = newDatacenter(cli, &dcs[i])

		err = cli.datacenters[i].scanHosts()
		if err != nil {
			return err
		}

		err = cli.datacenters[i].scanDatastores()
		if err != nil {
			return err
		}
	}
	return nil
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
		log.Fatalf("%s", err)
		return err
	}

	defer v.Destroy(cli.context)

	err = v.Retrieve(cli.context, []string{resType}, props, dst)
	if err != nil {
		log.Fatalf("%s", err)
		return err
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
		log.Errorf("pc.RetrieveOne fail %s", err)
		return err
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
	for i := 0; i < len(dcs); i += 1 {
		if dcs[i].GetId() == dcId {
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

	return NewHost(cli, &host, nil), nil
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
