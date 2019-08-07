package huawei

import (
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SDBInstanceAccount struct {
	multicloud.SDBInstanceAccountBase
	instance *SDBInstance
	Name     string
}

func (account *SDBInstanceAccount) GetId() string {
	return account.Name

}

func (account *SDBInstanceAccount) GetGlobalId() string {
	return account.Name
}

func (account *SDBInstanceAccount) GetName() string {
	return account.Name
}

func (account *SDBInstanceAccount) GetStatus() string {
	return api.DBINSTANCE_USER_AVAILABLE
}

func (account *SDBInstanceAccount) GetIDBInstanceAccountPrivileges() ([]cloudprovider.ICloudDBInstanceAccountPrivilege, error) {
	privileges, err := account.instance.region.GetDBInstancePrivvileges(account.instance.Id, account.Name)
	if err != nil {
		return nil, err
	}
	iprivileves := []cloudprovider.ICloudDBInstanceAccountPrivilege{}
	for i := 0; i < len(privileges); i++ {
		privileges[i].account = account
		iprivileves = append(iprivileves, &privileges[i])
	}
	return iprivileves, nil
}

func (region *SRegion) GetDBInstanceAccounts(instanceId string) ([]SDBInstanceAccount, error) {
	params := map[string]string{
		"instance_id": instanceId,
	}
	accounts := []SDBInstanceAccount{}
	err := doListAllWithPage(region.ecsClient.DBInstance.ListAccounts, params, &accounts)
	if err != nil {
		return nil, err
	}
	return accounts, nil
}

func (region *SRegion) GetDBInstancePrivvileges(instanceId string, username string) ([]SDatabasePrivilege, error) {
	params := map[string]string{
		"instance_id": instanceId,
		"user-name":   username,
	}
	privileges := []SDatabasePrivilege{}
	err := doListAllWithPage(region.ecsClient.DBInstance.ListPrivileges, params, &privileges)
	if err != nil {
		return nil, err
	}
	return privileges, nil
}
