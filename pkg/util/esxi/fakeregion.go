package esxi

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func (cli *SESXiClient) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SESXiClient) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SESXiClient) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	dcs, err := cli.GetDatacenters()
	if err != nil {
		return nil, err
	}

	ihosts := make([]cloudprovider.ICloudHost, 0)
	for i := 0; i < len(dcs); i += 1 {
		dcIHosts, err := dcs[i].GetIHosts()
		if err != nil {
			return nil, err
		}
		ihosts = append(ihosts, dcIHosts...)
	}
	return ihosts, nil
}

func (cli *SESXiClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return cli.FindHostByIp(id)
}

func (cli *SESXiClient) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	dcs, err := cli.GetDatacenters()
	if err != nil {
		return nil, err
	}

	iStorages := make([]cloudprovider.ICloudStorage, 0)
	for i := 0; i < len(dcs); i += 1 {
		dcIStorages, err := dcs[i].GetIStorages()
		if err != nil {
			return nil, err
		}
		iStorages = append(iStorages, dcIStorages...)
	}
	return iStorages, nil
}

func (cli *SESXiClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	iStorages, err := cli.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(iStorages); i += 1 {
		if iStorages[i].GetGlobalId() == id {
			return iStorages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (cli *SESXiClient) GetProvider() string {
	return models.CLOUD_PROVIDER_VMWARE
}
