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
	"os"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/s3cli"

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

	ownerId   string
	ownerName string

	iBuckets []cloudprovider.ICloudBucket

	client *s3cli.Client

	Debug bool
}

func NewObjectStoreClient(providerId string, providerName string, endpoint string, accessKey string, secret string, isDebug bool) (*SObjectStoreClient, error) {
	return NewObjectStoreClientAndFetch(providerId, providerName, endpoint, accessKey, secret, isDebug, true)
}

func NewObjectStoreClientAndFetch(providerId string, providerName string, endpoint string, accessKey string, secret string, isDebug bool, doFetch bool) (*SObjectStoreClient, error) {
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
	cli, err := s3cli.New(parts.Host, accessKey, secret, useSsl, client.Debug)
	if err != nil {
		return nil, errors.Wrap(err, "minio.New")
	}

	tr := httputils.GetTransport(true, time.Second*5)
	cli.SetCustomTransport(tr)

	client.client = cli
	client.SetVirtualObject(&client)

	if isDebug {
		cli.TraceOn(os.Stderr)
	}

	if doFetch {
		err = client.FetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}

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

func (cli *SObjectStoreClient) GetAccountId() string {
	return cli.ownerId
}

func (cli *SObjectStoreClient) GetIRegion() cloudprovider.ICloudRegion {
	return cli.GetVirtualObject().(cloudprovider.ICloudRegion)
}

func (cli *SObjectStoreClient) GetIBucketProvider() IBucketProvider {
	return cli.GetVirtualObject().(IBucketProvider)
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

func (cli *SObjectStoreClient) GetCloudEnv() string {
	return ""
}

////////////////////////////// IBucketProvider //////////////////////////////

func (cli *SObjectStoreClient) NewBucket(bucket s3cli.BucketInfo) cloudprovider.ICloudBucket {
	return &SBucket{
		client:    cli.GetIBucketProvider(),
		Name:      bucket.Name,
		CreatedAt: bucket.CreationDate,
	}
}

func (cli *SObjectStoreClient) GetEndpoint() string {
	return cli.endpoint
}

func (cli *SObjectStoreClient) S3Client() *s3cli.Client {
	return cli.client
}

func (cli *SObjectStoreClient) GetClientRC() map[string]string {
	return map[string]string{
		"S3_ACCESS_KEY": cli.accessKey,
		"S3_SECRET":     cli.secret,
		"S3_ACCESS_URL": cli.endpoint,
		"S3_BACKEND":    api.CLOUD_PROVIDER_GENERICS3,
	}
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

func (cli *SObjectStoreClient) ApplySnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
	return cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) CancelSnapshotPolicyToDisks(snapshotPolicyId string, diskId string) error {
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

func (cli *SObjectStoreClient) GetISkus() ([]cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (cli *SObjectStoreClient) CreateISku(name string, vCpu int, memoryMb int) error {
	return cloudprovider.ErrNotSupported
}

////////////////////////////////// S3 API ///////////////////////////////////

func (cli *SObjectStoreClient) GetIBuckets() ([]cloudprovider.ICloudBucket, error) {
	return cli.getIBuckets()
}

func (self *SObjectStoreClient) invalidateIBuckets() {
	self.iBuckets = nil
}

func (self *SObjectStoreClient) getIBuckets() ([]cloudprovider.ICloudBucket, error) {
	if self.iBuckets == nil {
		err := self.FetchBuckets()
		if err != nil {
			return nil, errors.Wrap(err, "fetchBuckets")
		}
	}
	return self.iBuckets, nil
}

func (cli *SObjectStoreClient) FetchBuckets() error {
	result, err := cli.client.ListBuckets()
	if err != nil {
		return errors.Wrap(err, "client.ListBuckets")
	}
	cli.ownerId = result.Owner.ID
	cli.ownerName = result.Owner.DisplayName
	buckets := result.Buckets.Bucket
	cli.iBuckets = make([]cloudprovider.ICloudBucket, len(buckets))
	for i := range buckets {
		b := cli.GetIBucketProvider().NewBucket(buckets[i])
		cli.iBuckets[i] = b
	}
	return nil
}

func (cli *SObjectStoreClient) CreateIBucket(name string, storageClass string, acl string) error {
	err := cli.client.MakeBucket(name, "")
	if err != nil {
		return errors.Wrap(err, "MakeBucket")
	}
	cli.invalidateIBuckets()
	return nil
}

func minioErrCode(err error) int {
	if srvErr, ok := err.(s3cli.ErrorResponse); ok {
		return srvErr.StatusCode
	}
	if srvErr, ok := err.(*s3cli.ErrorResponse); ok {
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
	cli.invalidateIBuckets()
	return nil
}

func (cli *SObjectStoreClient) GetIBucketLocation(name string) (string, error) {
	info, err := cli.client.GetBucketLocation(name)
	if err != nil {
		return "", errors.Wrap(err, "GetBucketLocation")
	}
	return info, nil
}

func (cli *SObjectStoreClient) GetIBucketWebsite(name string) (string, error) {
	info, err := cli.client.GetBucketWebsite(name)
	if err != nil {
		return "", errors.Wrap(err, "GetBucketWebsite")
	}
	return info, nil
}

func (cli *SObjectStoreClient) GetIBucketReferer(name string) (string, error) {
	info, err := cli.client.GetBucketReferer(name)
	if err != nil {
		return "", errors.Wrap(err, "GetBucketReferer")
	}
	return info, nil
}

func (cli *SObjectStoreClient) GetIBucketCors(name string) (string, error) {
	info, err := cli.client.GetBucketCors(name)
	if err != nil {
		return "", errors.Wrap(err, "GetBucketCors")
	}
	return info, nil
}

func (cli *SObjectStoreClient) GetIBucketLogging(name string) (*s3cli.BucketLoggingStatus, error) {
	info, err := cli.client.GetBucketLogging(name)
	if err != nil {
		return nil, errors.Wrap(err, "GetBucketLogging")
	}
	return info, nil
}

func (cli *SObjectStoreClient) SetIBucketLogging(name string, target string, targetPrefix string, email string) error {
	conf := s3cli.BucketLoggingStatus{}
	if len(target) > 0 {
		conf.LoggingEnabled.TargetBucket = target
		conf.LoggingEnabled.TargetPrefix = targetPrefix
		conf.LoggingEnabled.TargetGrants.Grant = []s3cli.Grant{
			{
				Grantee: s3cli.Grantee{
					Type:         s3cli.GRANTEE_TYPE_EMAIL,
					EmailAddress: email,
				},
				Permission: s3cli.PERMISSION_FULL_CONTROL,
			},
		}
	}
	err := cli.client.SetBucketLogging(name, conf)
	if err != nil {
		return errors.Wrap(err, "SetBucketLogging")
	}
	return nil
}

func (cli *SObjectStoreClient) GetIBucketInfo(name string) (string, error) {
	info, err := cli.client.GetBucketInfo(name)
	if err != nil {
		return "", errors.Wrap(err, "GetBucketInfo")
	}
	return info, nil
}

func (cli *SObjectStoreClient) GetIBucketAcl(name string) (cloudprovider.TBucketACLType, error) {
	acl, err := cli.client.GetBucketAcl(name)
	if err != nil {
		return "", errors.Wrap(err, "GetBucketAcl")
	}
	return cloudprovider.TBucketACLType(acl.GetCannedACL()), nil
}

func (cli *SObjectStoreClient) SetIBucketAcl(name string, cannedAcl cloudprovider.TBucketACLType) error {
	acl := s3cli.CannedAcl(cli.ownerId, cli.ownerName, string(cannedAcl))
	err := cli.client.SetBucketAcl(name, acl)
	if err != nil {
		return errors.Wrap(err, "SetBucketAcl")
	}
	return nil
}

func (cli *SObjectStoreClient) GetObjectAcl(bucket, key string) (cloudprovider.TBucketACLType, error) {
	acl, err := cli.client.GetObjectACL(bucket, key)
	if err != nil {
		return "", errors.Wrap(err, "GetBucketAcl")
	}
	return cloudprovider.TBucketACLType(acl.GetCannedACL()), nil
}

func (cli *SObjectStoreClient) SetObjectAcl(bucket, key string, cannedAcl cloudprovider.TBucketACLType) error {
	acl := s3cli.CannedAcl(cli.ownerId, cli.ownerName, string(cannedAcl))
	err := cli.client.SetObjectAcl(bucket, key, acl)
	if err != nil {
		return errors.Wrap(err, "SetObjectAcl")
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
	exist, header, err := cli.client.BucketExists(name)
	if err != nil {
		return false, errors.Wrap(err, "BucketExists")
	}
	if header != nil {
		log.Debugf("header: %s", jsonutils.Marshal(header))
	}
	return exist, nil
}

func (cli *SObjectStoreClient) GetIBucketById(name string) (cloudprovider.ICloudBucket, error) {
	return cloudprovider.GetIBucketById(cli, name)
}

func (cli *SObjectStoreClient) GetIBucketByName(name string) (cloudprovider.ICloudBucket, error) {
	return cli.GetIBucketById(name)
}

func (self *SObjectStoreClient) GetCapabilities() []string {
	caps := []string{
		// cloudprovider.CLOUD_CAPABILITY_PROJECT,
		// cloudprovider.CLOUD_CAPABILITY_COMPUTE,
		// cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
	}
	return caps
}
