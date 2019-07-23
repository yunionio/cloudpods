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

package objectstore

import (
	"net/url"
	"time"

	"github.com/minio/minio-go"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/object"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/httputils"
)

type SObjectStoreClient struct {
	object.SObject

	cloudprovider.SFakeOnPremiseRegion
	multicloud.SRegion

	providerId   string
	providerName string
	endpoint     string
	accessKey    string
	secret       string

	client *minio.Client

	Debug bool
}

func NewObjectStoreClient(providerId string, providerName string, endpoint string, accessKey string, secret string, isDebug bool) (*SObjectStoreClient, error) {
	client := SObjectStoreClient{
		providerId:   providerId,
		providerName: providerName,
		endpoint:     endpoint,
		accessKey:    accessKey,
		secret:       secret,
		Debug:        isDebug,
	}
	parts, err := url.Parse(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "url.Parse endpoint")
	}
	useSsl := false
	if parts.Scheme == "https" {
		useSsl = true
	}
	cli, err := minio.New(parts.Host, accessKey, secret, useSsl)
	if err != nil {
		return nil, errors.Wrap(err, "minio.New")
	}

	tr := httputils.GetTransport(true, time.Second*5)
	cli.SetCustomTransport(tr)

	client.client = cli

	return &client, nil
}

func (cli *SObjectStoreClient) GetSubAccounts() ([]cloudprovider.SSubAccount, error) {
	subAccount := cloudprovider.SSubAccount{
		Account:      cli.accessKey,
		Name:         cli.providerName,
		HealthStatus: api.CLOUD_PROVIDER_HEALTH_NORMAL,
	}
	return []cloudprovider.SSubAccount{subAccount}, nil
}

func (cli *SObjectStoreClient) GetIRegion() cloudprovider.ICloudRegion {
	return cli.GetVirtualObject().(cloudprovider.ICloudRegion)
}

func (cli *SObjectStoreClient) GetVersion() string {
	return ""
}

func (cli *SObjectStoreClient) About() jsonutils.JSONObject {
	about := jsonutils.NewDict()
	return about
}

func (cli *SObjectStoreClient) GetProvider() string {
	return api.CLOUD_PROVIDER_GENERICS3
}

///////////////////////////////// fake impletementations //////////////////////

func (cli *SObjectStoreClient) GetIZones() ([]cloudprovider.ICloudZone, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIVpcs() ([]cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIEips() ([]cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIVpcById(id string) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIZoneById(id string) (cloudprovider.ICloudZone, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIEipById(id string) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) DeleteSecurityGroup(vpcId, secgroupId string) error {
	return cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) SyncSecurityGroup(secgroupId string, vpcId string, name string, desc string, rules []secrules.SecurityRule) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) CreateIVpc(name string, desc string, cidr string) (cloudprovider.ICloudVpc, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) CreateEIP(eip *cloudprovider.SEip) (cloudprovider.ICloudEIP, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) CreateSnapshotPolicy(*cloudprovider.SnapshotPolicyInput) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) DeleteSnapshotPolicy(string) error {
	return cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskIds []string) error {
	return cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) CancelSnapshotPolicyToDisks(diskIds []string) error {
	return cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetISnapshotPolicies() ([]cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetISnapshotPolicyById(snapshotPolicyId string) (cloudprovider.ICloudSnapshotPolicy, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIStoragecaches() ([]cloudprovider.ICloudStoragecache, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetIStoragecacheById(id string) (cloudprovider.ICloudStoragecache, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetILoadBalancers() ([]cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetILoadBalancerAcls() ([]cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetILoadBalancerCertificates() ([]cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetILoadBalancerById(loadbalancerId string) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetILoadBalancerAclById(aclId string) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetILoadBalancerCertificateById(certId string) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancer) (cloudprovider.ICloudLoadbalancer, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) CreateILoadBalancerAcl(acl *cloudprovider.SLoadbalancerAccessControlList) (cloudprovider.ICloudLoadbalancerAcl, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) CreateILoadBalancerCertificate(cert *cloudprovider.SLoadbalancerCertificate) (cloudprovider.ICloudLoadbalancerCertificate, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetISkuById(skuId string) (cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) GetISkus(zoneId string) ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) CreateISku(sku *cloudprovider.SServerSku) (cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotSupported
}

////////////////////////////////// S3 API ///////////////////////////////////

func (cli *SObjectStoreClient) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	buckets, err := cli.client.ListBuckets()
	if err != nil {
		return nil, errors.Wrap(err, "client.ListBuckets")
	}
	ret := make([]cloudprovider.ICloudBucket, len(buckets))
	for i := range buckets {
		b := SBucket{
			client:    cli,
			Name:      buckets[i].Name,
			CreatedAt: buckets[i].CreationDate,
		}
		ret[i] = &b
	}
	return ret, nil
}

func (cli *SObjectStoreClient) CreateIBucket(name string, storageClass string, acl string) error {
	err := cli.client.MakeBucket(name, "")
	if err != nil {
		return errors.Wrap(err, "MakeBucket")
	}
	return nil
}

func minioErrCode(err error) int {
	if srvErr, ok := err.(minio.ErrorResponse); ok {
		return srvErr.StatusCode
	}
	if srvErr, ok := err.(*minio.ErrorResponse); ok {
		return srvErr.StatusCode
	}
	return -1
}

func (cli *SObjectStoreClient) DeleteIBucket(name string) error {
	err := cli.client.RemoveBucket(name)
	if err != nil {
		if minioErrCode(err) == 404 {
			return nil
		}
		return errors.Wrap(err, "RemoveBucket")
	}
	return nil
}

func (cli *SObjectStoreClient) GetIBucketPolicy(name string) (string, error) {
	policy, err := cli.client.GetBucketPolicy(name)
	if err != nil {
		return "", errors.Wrap(err, "GetBucketPolicy")
	}
	return policy, nil
}

func (cli *SObjectStoreClient) SetIBucketPolicy(name string, policy string) error {
	err := cli.client.SetBucketPolicy(name, policy)
	if err != nil {
		return errors.Wrap(err, "SetBucketPolicy")
	}
	return nil
}

func (cli *SObjectStoreClient) GetIBucketLiftcycle(name string) (string, error) {
	liftcycle, err := cli.client.GetBucketLifecycle(name)
	if err != nil {
		return "", errors.Wrap(err, "GetBucketLifecycle")
	}
	return liftcycle, nil
}

func (cli *SObjectStoreClient) IBucketExist(name string) (bool, error) {
	exist, err := cli.client.BucketExists(name)
	if err != nil {
		return false, errors.Wrap(err, "BucketExists")
	}
	return exist, nil
}

func (cli *SObjectStoreClient) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	return cloudprovider.GetIBucketById(cli, name)
}
