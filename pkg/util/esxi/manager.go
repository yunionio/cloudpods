package esxi

import (
	"context"
	"fmt"
	"net/url"
	"reflect"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"github.com/vmware/govmomi/session"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"strings"
)

const (
	CLOUD_PROVIDER_VMWARE = models.CLOUD_PROVIDER_VMWARE
)

type SESXiClient struct {
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

func (cli *SESXiClient) url() string {
	if cli.port == 443 || cli.port == 0 {
		return fmt.Sprintf("https://%s/sdk", cli.host)
	} else {
		return fmt.Sprintf("https://%s:%d/sdk", cli.host, cli.port)
	}
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

func (cli *SESXiClient) About() jsonutils.JSONObject {
	return jsonutils.Marshal(&cli.client.ServiceContent.About)
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

/*
func getStructFields(dst interface{}) []string {
	dataValue := reflect.Indirect(reflect.ValueOf(dst))
	dataType := dataValue.Type()
	if dataType.Kind() != reflect.Struct {
		log.Warningf("GetStructFeilds for non-struct data")
		return nil
	}
	return _getStructFields(dataType)
}

func _getStructFields(dataType reflect.Type) []string {
	ret := make([]string, 0)
	for i := 1; i < dataType.NumField(); i += 1 {
		field := dataType.Field(i)
		if field.Type.Kind() == reflect.Struct && field.Anonymous {
			subfields := _getStructFields(field.Type)
			ret = append(ret, subfields...)
		} else if gotypes.IsFieldExportable(field.Name)  {
			log.Debugf("%s: %s", field.Name, field.Type.Name())
			ret = append(ret, utils.CamelSplit(field.Name, "_"))
		}
	}
	return ret
}
*/

func (cli *SESXiClient) references2Objects(refs []types.ManagedObjectReference, props []string, dst interface{}) error {
	pc := property.DefaultCollector(cli.client.Client)
	err := pc.Retrieve(cli.context, refs, []string{"name", "config", "summary"}, dst)
	if err != nil {
		return err
	}
	return nil
}

func (cli *SESXiClient) reference2Object(ref types.ManagedObjectReference, props []string, dst interface{}) error {
	pc := property.DefaultCollector(cli.client.Client)
	return pc.RetrieveOne(cli.context, ref, props, dst)
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

func (cli *SESXiClient) FindDatacenterById(dcId string) (*SDatacenter, error) {
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

func (cli *SESXiClient) getPrivateId(idStr string) string {
	if strings.HasPrefix(idStr, cli.providerId) {
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
