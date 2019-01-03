package huawei

import (
	"fmt"
	"strings"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/huawei/client"
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth"
	"yunion.io/x/onecloud/pkg/util/huawei/client/auth/credentials"
)

/*
待解决问题：
1.同步的子账户中有一条空记录.需要查原因
2.安全组同步需要进一步确认
3.实例接口需要进一步确认
*/

const (
	CLOUD_PROVIDER_HUAWEI    = models.CLOUD_PROVIDER_HUAWEI
	CLOUD_PROVIDER_HUAWEI_CN = "华为云"

	HUAWEI_DEFAULT_REGION = "cn-hangzhou"
	HUAWEI_API_VERSION    = "2018-12-25"
)

type SHuaweiClient struct {
	signer auth.Signer

	providerId   string
	providerName string
	projectId    string // 华为云项目ID.
	accessUrl    string // 服务区域 ChinaCloud | InternationalCloud
	accessKey    string
	secret       string
	iregions     []cloudprovider.ICloudRegion
}

func parseAccount(account string) (accessKey string, projectId string) {
	segs := strings.Split(account, "/")
	if len(segs) == 2 {
		accessKey = segs[0]
		projectId = segs[1]
	} else {
		accessKey = account
		projectId = ""
	}

	return
}

// 进行资源操作时参数account 对应数据库cloudprovider表中的account字段,由accessKey和projectID两部分组成，通过"/"分割。
// 初次导入Subaccount时，参数account对应cloudaccounts表中的account字段，即accesskey。此时projectID为空，
// 只能进行同步子账号、查询region列表等projectId无关的操作。
// todo: 通过accessurl支持国际站。目前暂时未支持国际站
func NewHuaweiClient(providerId, providerName, accessurl, account, secret string) (*SHuaweiClient, error) {
	accessKey, projectId := parseAccount(account)
	client := SHuaweiClient{
		providerId:   providerId,
		providerName: providerName,
		projectId:    projectId,
		accessKey:    accessKey,
		secret:       secret,
	}
	err := client.fetchRegions()
	if err != nil {
		return nil, err
	}

	cred := credentials.NewAccessKeyCredential(client.accessKey, client.accessKey)
	client.signer, err = auth.NewSignerWithCredential(cred)
	if err != nil {
		return nil, err
	}
	return &client, nil
}

func (self *SHuaweiClient) fetchRegions() error {
	huawei, _ := clients.NewClientWithAccessKey("", "", self.accessKey, self.secret)
	regions := make([]SRegion, 0)
	err := DoList(huawei.Regions.List, nil, &regions)
	if err != nil {
		return err
	}

	filtedRegions := make([]SRegion, 0)
	if len(self.projectId) > 0 {
		project, err := self.GetProjectById(self.projectId)
		if err != nil {
			return err
		}

		regionId := strings.Split(project.Name, "_")[0]
		for _, region := range regions {
			if region.ID == regionId {
				filtedRegions = append(filtedRegions, region)
			}
		}
	} else {
		filtedRegions = regions
	}

	self.iregions = make([]cloudprovider.ICloudRegion, len(filtedRegions))
	for i := 0; i < len(filtedRegions); i += 1 {
		filtedRegions[i].client = self
		_, err := filtedRegions[i].getECSClient()
		if err != nil {
			return err
		}
		self.iregions[i] = &filtedRegions[i]
	}
	return nil
}

func (self *SHuaweiClient) UpdateAccount(accessKey, secret string) error {
	if self.accessKey != accessKey || self.secret != secret {
		self.accessKey = accessKey
		self.secret = secret
		return self.fetchRegions()
	} else {
		return nil
	}
}

func (self *SHuaweiClient) GetRegions() []SRegion {
	regions := make([]SRegion, len(self.iregions))
	for i := 0; i < len(regions); i += 1 {
		region := self.iregions[i].(*SRegion)
		regions[i] = *region
	}
	return regions
}

func (self *SHuaweiClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	projects, err := self.fetchProjects()
	if err != nil {
		return nil, err
	}

	// https://support.huaweicloud.com/api-iam/zh-cn_topic_0074171149.html
	subAccounts := make([]cloudprovider.SSubAccount, 0)
	for i := range projects {
		project := projects[i]
		// name 为MOS的project是华为云内部的一个特殊project。不需要同步到本地
		if strings.ToLower(project.Name) == "mos" {
			continue
		}
		s := cloudprovider.SSubAccount{
			Name:         project.Name,
			State:        models.CLOUD_PROVIDER_CONNECTED,
			Account:      fmt.Sprintf("%s/%s", self.accessKey, project.ID),
			HealthStatus: project.GetHealthStatus(),
		}

		subAccounts = append(subAccounts, s)
	}

	return subAccounts, nil
}

func (self *SHuaweiClient) GetIRegions() []cloudprovider.ICloudRegion {
	return self.iregions
}

func (self *SHuaweiClient) GetIRegionById(id string) (cloudprovider.ICloudRegion, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetGlobalId() == id {
			return self.iregions[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SHuaweiClient) GetRegion(regionId string) *SRegion {
	if len(regionId) == 0 {
		regionId = HUAWEI_DEFAULT_REGION
	}
	for i := 0; i < len(self.iregions); i += 1 {
		if self.iregions[i].GetId() == regionId {
			return self.iregions[i].(*SRegion)
		}
	}
	return nil
}

func (self *SHuaweiClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ihost, err := self.iregions[i].GetIHostById(id)
		if err == nil {
			return ihost, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SHuaweiClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		ivpc, err := self.iregions[i].GetIVpcById(id)
		if err == nil {
			return ivpc, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SHuaweiClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	for i := 0; i < len(self.iregions); i += 1 {
		istorage, err := self.iregions[i].GetIStorageById(id)
		if err == nil {
			return istorage, nil
		} else if err != cloudprovider.ErrNotFound {
			return nil, err
		}
	}
	return nil, cloudprovider.ErrNotFound
}

type SAccountBalance struct {
	AvailableAmount float64
}

func (self *SHuaweiClient) QueryAccountBalance() (*SAccountBalance, error) {
	// todo: implement me
	return nil, nil
}

func (self *SHuaweiClient) GetVersion() string {
	return HUAWEI_API_VERSION
}
