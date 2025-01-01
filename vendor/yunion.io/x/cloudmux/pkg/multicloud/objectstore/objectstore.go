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
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/object"
	"yunion.io/x/pkg/util/httputils"
	"yunion.io/x/pkg/util/secrules"
	"yunion.io/x/s3cli"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type S3SignVersion string

const (
	S3SignAlgDefault = S3SignVersion("")
	S3SignAlgV4      = S3SignVersion("v4")
	S3SignAlgV2      = S3SignVersion("v2")
)

type ObjectStoreClientConfig struct {
	cpcfg cloudprovider.ProviderConfig

	endpoint     string
	accessKey    string
	accessSecret string

	signVer S3SignVersion

	debug bool
}

func NewObjectStoreClientConfig(endpoint, accessKey, accessSecret string) *ObjectStoreClientConfig {
	cfg := &ObjectStoreClientConfig{
		endpoint:     endpoint,
		accessKey:    accessKey,
		accessSecret: accessSecret,
	}
	return cfg
}

func (cfg *ObjectStoreClientConfig) CloudproviderConfig(cpcfg cloudprovider.ProviderConfig) *ObjectStoreClientConfig {
	cfg.cpcfg = cpcfg
	return cfg
}

func (cfg *ObjectStoreClientConfig) SignVersion(signVer S3SignVersion) *ObjectStoreClientConfig {
	cfg.signVer = signVer
	return cfg
}

func (cfg *ObjectStoreClientConfig) Debug(debug bool) *ObjectStoreClientConfig {
	cfg.debug = debug
	return cfg
}

func (cfg *ObjectStoreClientConfig) GetCloudproviderConfig() cloudprovider.ProviderConfig {
	return cfg.cpcfg
}

func (cfg *ObjectStoreClientConfig) GetEndpoint() string {
	return cfg.endpoint
}

func (cfg *ObjectStoreClientConfig) GetAccessKey() string {
	return cfg.accessKey
}

func (cfg *ObjectStoreClientConfig) GetAccessSecret() string {
	return cfg.accessSecret
}

func (cfg *ObjectStoreClientConfig) GetDebug() bool {
	return cfg.debug
}

type SObjectStoreClient struct {
	object.SObject

	*ObjectStoreClientConfig

	cloudprovider.SFakeOnPremiseRegion
	multicloud.SRegion

	ownerId   string
	ownerName string

	iBuckets []cloudprovider.ICloudBucket

	client *s3cli.Client
}

func NewObjectStoreClient(cfg *ObjectStoreClientConfig) (*SObjectStoreClient, error) {
	return NewObjectStoreClientAndFetch(cfg, true)
}

func NewObjectStoreClientAndFetch(cfg *ObjectStoreClientConfig, doFetch bool) (*SObjectStoreClient, error) {
	client := SObjectStoreClient{
		ObjectStoreClientConfig: cfg,
	}
	parts, err := url.Parse(cfg.endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "url.Parse endpoint")
	}
	useSsl := false
	if parts.Scheme == "https" {
		useSsl = true
	}
	s3cliNewFunc := s3cli.New

	switch cfg.signVer {
	case S3SignAlgV4:
		log.Debugf("Use v4 signing algorithm")
		s3cliNewFunc = s3cli.NewV4
	case S3SignAlgV2:
		log.Debugf("Use v2 signing algorithm")
		s3cliNewFunc = s3cli.NewV2
	default:
		log.Debugf("s3 sign algirithm version not set, use default")
	}

	cli, err := s3cliNewFunc(
		parts.Host,
		client.accessKey,
		client.accessSecret,
		useSsl,
		client.debug,
	)
	if err != nil {
		return nil, errors.Wrap(err, "minio.New")
	}

	tr := httputils.GetTransport(true)
	tr.Proxy = cfg.cpcfg.ProxyFunc
	tr.DialContext = (&net.Dialer{
		Timeout:   60 * time.Second,
		KeepAlive: 60 * time.Second,
		DualStack: true,
	}).DialContext
	tr.IdleConnTimeout = 90 * time.Second
	cli.SetCustomTransport(tr)

	client.client = cli
	client.SetVirtualObject(&client)

	if client.debug {
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
		Id:           cli.GetAccountId(),
		Account:      cli.accessKey,
		Name:         cli.cpcfg.Name,
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

func (cli *SObjectStoreClient) GetI18n() cloudprovider.SModelI18nTable {
	table := cloudprovider.SModelI18nTable{}
	table["name"] = cloudprovider.NewSModelI18nEntry(cli.GetName()).CN(cli.GetName())
	return table
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
		"S3_SECRET":     cli.accessSecret,
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

func (cli *SObjectStoreClient) CreateIVpc(opts *cloudprovider.VpcCreateOptions) (cloudprovider.ICloudVpc, error) {
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

func (cli *SObjectStoreClient) CreateILoadBalancer(loadbalancer *cloudprovider.SLoadbalancerCreateOptions) (cloudprovider.ICloudLoadbalancer, error) {
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

func (cli *SObjectStoreClient) CreateISku(opts *cloudprovider.SServerSkuCreateOption) (cloudprovider.ICloudSku, error) {
	return nil, cloudprovider.ErrNotSupported
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
		if strings.Contains(err.Error(), "not implemented") {
			// ignore not implemented error
			return nil // cloudprovider.ErrNotImplemented
		} else {
			return errors.Wrap(err, "SetBucketAcl")
		}
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
		// cloudprovider.CLOUD_CAPABILITY_NETWORK,
		// cloudprovider.CLOUD_CAPABILITY_LOADBALANCER,
		cloudprovider.CLOUD_CAPABILITY_OBJECTSTORE,
		// cloudprovider.CLOUD_CAPABILITY_RDS,
		// cloudprovider.CLOUD_CAPABILITY_CACHE,
		// cloudprovider.CLOUD_CAPABILITY_EVENT,
	}
	return caps
}
