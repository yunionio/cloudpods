package models

import (
	"fmt"
	"strings"

	"yunion.io/x/onecloud/pkg/appctx"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SManagedResourceBase struct {
	ManagerId string `width:"128" charset:"ascii" nullable:"true" list:"admin" create:"admin_optional"` // Column(VARCHAR(ID_LENGTH, charset='ascii'), nullable=True)
}

func (self *SManagedResourceBase) GetCloudprovider() *SCloudprovider {
	if len(self.ManagerId) > 0 {
		return CloudproviderManager.FetchCloudproviderById(self.ManagerId)
	}
	return nil
}

func (self *SManagedResourceBase) GetCloudaccount() *SCloudaccount {
	cp := self.GetCloudprovider()
	if cp == nil {
		return nil
	}
	return cp.GetCloudaccount()
}

func (self *SManagedResourceBase) GetDriver() (cloudprovider.ICloudProvider, error) {
	provider := self.GetCloudprovider()
	if provider == nil {
		if len(self.ManagerId) > 0 {
			return nil, cloudprovider.ErrInvalidProvider
		}
		return nil, fmt.Errorf("Resource is self managed")
	}
	return provider.GetDriver()
}

func (self *SManagedResourceBase) GetProviderName() string {
	driver := self.GetCloudprovider()
	if driver != nil {
		return driver.Provider
	}
	return ""
}

func (self *SManagedResourceBase) IsManaged() bool {
	return len(self.ManagerId) > 0
}

/*func (self *SManagedResourceBase) getExtraDetails(ctx context.Context, extra *jsonutils.JSONDict) *jsonutils.JSONDict {
	manager := self.GetCloudprovider()
	if manager != nil {
		extra.Add(jsonutils.NewString(manager.Name), "manager")
		extra.Add(jsonutils.NewString(manager.ProjectId), "manager_tenant_id")
		extra.Add(jsonutils.NewString(manager.ProjectId), "manager_project_id")
		project := manager.getProject(ctx)
		if project != nil {
			extra.Add(jsonutils.NewString(project.Name), "manager_tenant")
			extra.Add(jsonutils.NewString(project.Name), "manager_project")
		}
	}
	return extra
}
*/

type SCloudProviderInfo struct {
	Provider         string `json:",omitempty"`
	Account          string `json:",omitempty"`
	AccountId        string `json:",omitempty"`
	Manager          string `json:",omitempty"`
	ManagerId        string `json:",omitempty"`
	ManagerProject   string `json:",omitempty"`
	ManagerProjectId string `json:",omitempty"`
	Region           string `json:",omitempty"`
	RegionId         string `json:",omitempty"`
	RegionExtId      string `json:",omitempty"`
	Zone             string `json:",omitempty"`
	ZoneId           string `json:",omitempty"`
	ZoneExtId        string `json:",omitempty"`
}

func fetchExternalId(extId string) string {
	pos := strings.LastIndexByte(extId, '/')
	if pos > 0 {
		return extId[pos+1:]
	} else {
		return extId
	}
}

func MakeCloudProviderInfo(region *SCloudregion, zone *SZone, provider *SCloudprovider) SCloudProviderInfo {
	info := SCloudProviderInfo{}

	if zone != nil {
		info.Zone = zone.GetName()
		info.ZoneId = zone.GetId()
	}

	if region != nil {
		info.Region = region.GetName()
		info.RegionId = region.GetId()
	}

	if provider != nil {
		info.Manager = provider.GetName()
		info.ManagerId = provider.GetId()

		if len(provider.ProjectId) > 0 {
			info.ManagerProjectId = provider.ProjectId
			tc, err := db.TenantCacheManager.FetchTenantById(appctx.Background, provider.ProjectId)
			if err == nil {
				info.ManagerProject = tc.GetName()
			}
		}

		account := provider.GetCloudaccount()
		info.Account = account.GetName()
		info.AccountId = account.GetId()

		info.Provider = provider.Provider

		if region != nil {
			info.RegionExtId = fetchExternalId(region.ExternalId)
			if zone != nil {
				info.ZoneExtId = fetchExternalId(zone.ExternalId)
			}
		}
	}

	return info
}
