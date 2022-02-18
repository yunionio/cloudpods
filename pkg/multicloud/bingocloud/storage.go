/*
 * @Author: your name
 * @Date: 2022-02-17 21:54:45
 * @LastEditTime: 2022-02-18 12:55:33
 * @LastEditors: Please set LastEditors
 * @Description: 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
 * @FilePath: \cloudpods\pkg\multicloud\bingocloud\storage.go
 */
package bingocloud

import (
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SStorage struct {
	Disabled     bool   `json:"disabled"`
	DrCloudId    string `json:"drCloudId"`
	ParameterSet struct {
		Item struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"item"`
	} `json:"parameterSet"`
	StorageId    string `json:"storageId"`
	ClusterId    string `json:"clusterId"`
	UsedBy       string `json:"usedBy"`
	SpaceMax     string `json:"spaceMax"`
	IsDRStorage  string `json:"isDRStorage"`
	ScheduleTags string `json:"scheduleTags"`
	StorageName  string `json:"storageName"`
	Location     string `json:"location"`
	SpaceUsed    string `json:"spaceUsed"`
	StorageType  string `json:"storageType"`
	FileFormat   string `json:"fileFormat"`
	ResUsage     string `json:"resUsage"`
}

func (self *SRegion) GetStorages() ([]SStorage, error) {
	resp, err := self.invoke("DescribeStorages", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		StorageSet struct {
			Item []SStorage
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return result.StorageSet.Item, nil
}

func (self *SRegion) GetStorage(id string) (*SStorage, error) {
	storage := &SStorage{}
	// return storage, self.get("storage_containers", id, nil, storage)
	return storage, cloudprovider.ErrNotImplemented
}
