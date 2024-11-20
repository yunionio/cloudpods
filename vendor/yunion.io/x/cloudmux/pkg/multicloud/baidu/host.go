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

package baidu

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"net/url"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := host.zone.region.GetInstances("", []string{})
	if err != nil {
		return nil, err
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i += 1 {
		vms[i].host = host
		ivms[i] = &vms[i]
	}
	return ivms, nil
}

func (host *SHost) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vm, err := host.zone.region.CreateInstance(host.zone.ZoneName, opts)
	if err != nil {
		return nil, err
	}
	vm.host = host
	return vm, nil
}

func PKCS7Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

type ecb struct {
	b         cipher.Block
	blockSize int
}

func newECB(b cipher.Block) *ecb {
	return &ecb{
		b:         b,
		blockSize: b.BlockSize(),
	}
}

type ecbEncrypter ecb

func NewECBEncrypter(b cipher.Block) cipher.BlockMode {
	return (*ecbEncrypter)(newECB(b))
}

func (x *ecbEncrypter) BlockSize() int { return x.blockSize }

func (x *ecbEncrypter) CryptBlocks(dst, src []byte) {
	if len(src)%x.blockSize != 0 {
		panic("crypto/cipher: input not full blocks")
	}
	if len(dst) < len(src) {
		panic("crypto/cipher: output smaller than input")
	}
	for len(src) > 0 {
		x.b.Encrypt(dst, src[:x.blockSize])
		src = src[x.blockSize:]
		dst = dst[x.blockSize:]
	}
}

func AesECBEncryptHex(key, message string) (string, error) {
	// ECB is left out intentionally because it's insecure, check https://github.com/golang/go/issues/5597
	if len(key) < 16 {
		return "", fmt.Errorf("Invalid SecretKey")
	}
	keyBytes := []byte(key[:16])
	msgBytes := []byte(message)
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", err
	}
	blockSize := block.BlockSize()
	msgBytes = PKCS7Padding(msgBytes, blockSize)
	blockMode := NewECBEncrypter(block)
	crypted := make([]byte, len(msgBytes))
	blockMode.CryptBlocks(crypted, msgBytes)
	return hex.EncodeToString(crypted), nil
}

func (region *SRegion) CreateInstance(zoneName string, opts *cloudprovider.SManagedVMCreateConfig) (*SInstance, error) {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	tags := []BaiduTag{}
	for k, v := range opts.Tags {
		tags = append(tags, BaiduTag{
			TagKey:   k,
			TagValue: v,
		})
	}
	billing := map[string]interface{}{
		"paymentTiming": "Postpaid",
	}
	if opts.BillingCycle != nil {
		billing["paymentTiming"] = "Prepaid"
		reservation := map[string]interface{}{}
		if opts.BillingCycle.GetYears() > 0 {
			reservation["reservationTimeUnit"] = "year"
			reservation["reservationLength"] = opts.BillingCycle.GetYears()
		} else if opts.BillingCycle.GetMonths() > 0 {
			reservation["reservationTimeUnit"] = "month"
			reservation["reservationLength"] = opts.BillingCycle.GetMonths()
		}
		billing["reservation"] = reservation
	}

	disks := []map[string]interface{}{}
	for _, disk := range opts.DataDisks {
		disks = append(disks, map[string]interface{}{
			"cdsSizeInGB": disk.SizeGB,
			"storageType": disk.StorageType,
		})
	}

	body := map[string]interface{}{
		"imageId":               opts.ExternalImageId,
		"spec":                  opts.InstanceType,
		"rootDiskSizeInGb":      opts.SysDisk.SizeGB,
		"rootDiskStorageType":   opts.SysDisk.StorageType,
		"networkCapacityInMbps": opts.PublicIpBw,
		"name":                  opts.Name,
		"hostname":              opts.Hostname,
		"adminPass":             opts.Password,
		"zoneName":              zoneName,
		"subnetId":              opts.ExternalNetworkId,
		"tags":                  tags,
		"userData":              opts.UserData,
		"billing":               billing,
		"createCdsList":         disks,
	}
	if len(opts.Password) > 0 {
		var err error
		body["adminPass"], err = AesECBEncryptHex(region.client.accessKeySecret, opts.Password)
		if err != nil {
			return nil, errors.Wrapf(err, "AesECBEncryptHex")
		}
	}

	if len(opts.IpAddr) > 0 {
		body["internalIps"] = []string{opts.IpAddr}
	}

	if len(opts.PublicKey) > 0 {
		keypair, err := region.SyncKeypair(opts.KeypairName, opts.PublicKey)
		if err != nil {
			return nil, err
		}
		body["keypairId"] = keypair.KeypairId
	}

	if len(opts.ExternalSecgroupIds) > 0 {
		body["securityGroupId"] = opts.ExternalSecgroupIds[0]
	}
	resp, err := region.bccPost("v2/instanceBySpec", params, body)
	if err != nil {
		return nil, err
	}
	ret := struct {
		InstanceIds []string
		WarningList []string
	}{}
	err = resp.Unmarshal(&ret)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s", resp.String())
	}
	for _, vmId := range ret.InstanceIds {
		return region.GetInstance(vmId)
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, "after create %s", resp.String())
}

func (host *SHost) GetAccessIp() string {
	return ""
}

func (host *SHost) GetAccessMac() string {
	return ""
}

func (host *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", host.zone.region.client.cpcfg.Name, host.zone.GetId())
}

func (host *SHost) GetNodeCount() int8 {
	return 0
}

func (host *SHost) GetSN() string {
	return ""
}

func (host *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (host *SHost) GetCpuCount() int {
	return 0
}

func (host *SHost) GetCpuDesc() string {
	return ""
}

func (host *SHost) GetCpuMhz() int {
	return 0
}

func (host *SHost) GetMemSizeMB() int {
	return 0
}

func (host *SHost) GetStorageSizeMB() int64 {
	return 0
}

func (host *SHost) GetStorageClass() string {
	return ""
}

func (host *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (host *SHost) GetEnabled() bool {
	return true
}

func (host *SHost) GetIsMaintenance() bool {
	return false
}

func (host *SHost) IsEmulated() bool {
	return true
}

func (host *SHost) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", host.zone.region.client.cpcfg.Id, host.zone.GetId())
}

func (host *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", host.zone.region.client.cpcfg.Id, host.zone.GetId())
}

func (host *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (host *SHost) GetHostType() string {
	return api.HOST_TYPE_BAIDU
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.zone.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (host *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorages()
}

func (host *SHost) GetIVMById(vmId string) (cloudprovider.ICloudVM, error) {
	vm, err := host.zone.region.GetInstance(vmId)
	if err != nil {
		return nil, err
	}
	vm.host = host
	return vm, nil
}

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_BAIDU_CN), "manufacture")
	return info
}

func (host *SHost) GetVersion() string {
	return ""
}
