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

package esxi

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/progress"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/vmdkutils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

var DATASTORE_PROPS = []string{"name", "parent", "info", "summary", "host", "vm"}

type SDatastore struct {
	multicloud.SStorageBase
	SManagedObject

	// vms []cloudprovider.ICloudVM

	ihosts []cloudprovider.ICloudHost
	idisks []cloudprovider.ICloudDisk

	storageCache *SDatastoreImageCache
}

func NewDatastore(manager *SESXiClient, ds *mo.Datastore, dc *SDatacenter) *SDatastore {
	return &SDatastore{SManagedObject: newManagedObject(manager, ds, dc)}
}

func (self *SDatastore) getDatastore() *mo.Datastore {
	return self.object.(*mo.Datastore)
}

func (self *SDatastore) GetGlobalId() string {
	volId, err := self.getVolumeId()
	if err != nil {
		log.Errorf("get datastore global ID error %s", err)
	}
	return volId
}

func (self *SDatastore) GetName() string {
	volName, err := self.getVolumeName()
	if err != nil {
		log.Fatalf("datastore get name error %s", err)
	}
	return fmt.Sprintf("%s-%s", self.getVolumeType(), volName)
}

func (self *SDatastore) GetRelName() string {
	return self.getDatastore().Info.GetDatastoreInfo().Name
}

func (self *SDatastore) GetCapacityMB() int64 {
	moStore := self.getDatastore()
	return moStore.Summary.Capacity / 1024 / 1024
}

func (self *SDatastore) GetCapacityUsedMB() int64 {
	moStore := self.getDatastore()
	return self.GetCapacityMB() - moStore.Summary.FreeSpace/1024/1024
}

func (self *SDatastore) GetCapacityFreeMB() int64 {
	moStore := self.getDatastore()
	return moStore.Summary.FreeSpace / 1024 / 1024
}

func (self *SDatastore) GetEnabled() bool {
	return true
}

func (self *SDatastore) GetStatus() string {
	if self.getDatastore().Summary.Accessible {
		return api.STORAGE_ONLINE
	} else {
		return api.STORAGE_OFFLINE
	}
}

func (self *SDatastore) Refresh() error {
	base := self.SManagedObject
	var moObj mo.Datastore
	err := self.manager.reference2Object(self.object.Reference(), DATASTORE_PROPS, &moObj)
	if err != nil {
		return err
	}
	base.object = &moObj
	*self = SDatastore{}
	self.SManagedObject = base
	return nil
}

func (self *SDatastore) getVolumeId() (string, error) {
	moStore := self.getDatastore()
	switch fsInfo := moStore.Info.(type) {
	case *types.VmfsDatastoreInfo:
		if fsInfo.Vmfs.Local == nil || *fsInfo.Vmfs.Local {
			host, err := self.getLocalHost()
			if err == nil {
				return fmt.Sprintf("%s:%s", host.GetAccessIp(), fsInfo.Vmfs.Uuid), nil
			}
		}
		return fsInfo.Vmfs.Uuid, nil
	case *types.NasDatastoreInfo:
		return fmt.Sprintf("%s:%s", fsInfo.Nas.RemoteHost, fsInfo.Nas.RemotePath), nil
	}
	if moStore.Summary.Type == "vsan" {
		vsanId := moStore.Summary.Url
		vsanId = vsanId[strings.Index(vsanId, "vsan:"):]
		endIdx := len(vsanId)
		for ; endIdx >= 0 && vsanId[endIdx-1] == '/'; endIdx -= 1 {
		}
		return vsanId[:endIdx], nil
	}
	log.Fatalf("unsupported volume type %#v", moStore.Info)
	return "", cloudprovider.ErrNotImplemented
}

func (self *SDatastore) getVolumeType() string {
	return self.getDatastore().Summary.Type
}

func (self *SDatastore) getVolumeName() (string, error) {
	moStore := self.getDatastore()

	if self.isLocalVMFS() {
		host, err := self.getLocalHost()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s-%s", host.GetAccessIp(), moStore.Info.GetDatastoreInfo().Name), nil
	}
	dc, err := self.GetDatacenter()
	if err != nil {
		return "", nil
	}
	return fmt.Sprintf("%s-%s", dc.GetName(), moStore.Info.GetDatastoreInfo().Name), nil
}

func (self *SDatastore) getAttachedHosts() ([]cloudprovider.ICloudHost, error) {
	ihosts := make([]cloudprovider.ICloudHost, 0)

	moStore := self.getDatastore()
	for i := 0; i < len(moStore.Host); i += 1 {
		idstr := moRefId(moStore.Host[i].Key)
		host, err := self.datacenter.GetIHostByMoId(idstr)
		if err != nil {
			return nil, err
		}
		ihosts = append(ihosts, host)
	}

	return ihosts, nil
}

func (self *SDatastore) getCachedAttachedHosts() ([]cloudprovider.ICloudHost, error) {
	if self.ihosts == nil {
		var err error
		self.ihosts, err = self.getAttachedHosts()
		if err != nil {
			return nil, err
		}
	}
	return self.ihosts, nil
}

func (self *SDatastore) GetAttachedHosts() ([]cloudprovider.ICloudHost, error) {
	return self.getCachedAttachedHosts()
}

func (self *SDatastore) getLocalHost() (cloudprovider.ICloudHost, error) {
	hosts, err := self.GetAttachedHosts()
	if err != nil {
		return nil, errors.Wrap(err, "self.GetAttachedHosts")
	}
	if len(hosts) == 1 {
		return hosts[0], nil
	}
	return nil, cloudprovider.ErrInvalidStatus
}

func (self *SDatastore) GetIStoragecache() cloudprovider.ICloudStoragecache {
	if self.isLocalVMFS() {
		ihost, err := self.getLocalHost()
		if err != nil {
			log.Errorf("GetIStoragecache getLocalHost fail %s", err)
			return nil
		}
		host := ihost.(*SHost)
		sc, err := host.getLocalStorageCache()
		if err != nil {
			log.Errorf("GetIStoragecache getLocalStorageCache fail %s", err)
			return nil
		}
		return sc
	} else {
		return self.getStorageCache()
	}
}

func (self *SDatastore) getStorageCache() *SDatastoreImageCache {
	if self.storageCache == nil {
		self.storageCache = &SDatastoreImageCache{
			datastore: self,
		}
	}
	return self.storageCache
}

func (self *SDatastore) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (self *SDatastore) FetchNoTemplateVMs() ([]*SVirtualMachine, error) {
	mods := self.getDatastore()
	filter := property.Filter{}
	filter["datastore"] = mods.Reference()
	return self.datacenter.fetchVMsWithFilter(filter)
}

func (self *SDatastore) FetchTemplateVMs() ([]*SVirtualMachine, error) {
	mods := self.getDatastore()
	filter := property.Filter{}
	filter["config.template"] = true
	filter["datastore"] = mods.Reference()
	return self.datacenter.fetchVMsWithFilter(filter)
}

func (self *SDatastore) FetchTemplateVMById(id string) (*SVirtualMachine, error) {
	mods := self.getDatastore()
	filter := property.Filter{}
	uuid := toTemplateUuid(id)
	filter["summary.config.uuid"] = uuid
	filter["config.template"] = true
	filter["datastore"] = mods.Reference()
	vms, err := self.datacenter.fetchVMsWithFilter(filter)
	if err != nil {
		return nil, err
	}
	if len(vms) == 0 {
		return nil, errors.ErrNotFound
	}
	return vms[0], nil
}

func (self *SDatastore) FetchFakeTempateVMById(id string, regex string) (*SVirtualMachine, error) {
	mods := self.getDatastore()
	filter := property.Filter{}
	uuid := toTemplateUuid(id)
	filter["summary.config.uuid"] = uuid
	filter["datastore"] = mods.Reference()
	filter["summary.runtime.powerState"] = types.VirtualMachinePowerStatePoweredOff
	filter["config.template"] = false
	movms, err := self.datacenter.fetchMoVms(filter, []string{"name"})
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch mo.VirtualMachines")
	}
	vms, err := self.datacenter.fetchFakeTemplateVMs(movms, regex)
	if err != nil {
		return nil, err
	}
	if len(vms) == 0 {
		return nil, errors.ErrNotFound
	}
	return vms[0], nil
}

func (self *SDatastore) FetchFakeTempateVMs(regex string) ([]*SVirtualMachine, error) {
	mods := self.getDatastore()
	filter := property.Filter{}
	filter["datastore"] = mods.Reference()
	filter["summary.runtime.powerState"] = types.VirtualMachinePowerStatePoweredOff
	filter["config.template"] = false
	movms, err := self.datacenter.fetchMoVms(filter, []string{"name"})
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch mo.VirtualMachines")
	}
	return self.datacenter.fetchFakeTemplateVMs(movms, regex)
}

func (self *SDatastore) getVMs() ([]cloudprovider.ICloudVM, error) {
	dc, err := self.GetDatacenter()
	if err != nil {
		return nil, errors.Wrapf(err, "GetDatacenter")
	}
	vms := self.getDatastore().Vm
	if len(vms) == 0 {
		return nil, nil
	}
	svms, err := dc.fetchVmsFromCache(vms)
	if err != nil {
		return nil, errors.Wrapf(err, "fetchVmsFromCache")
	}
	ret := make([]cloudprovider.ICloudVM, len(svms))
	for i := range svms {
		ret[i] = svms[i]
	}
	return ret, err
}

func (self *SDatastore) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	vms, err := self.getVMs()
	if err != nil {
		return nil, errors.Wrapf(err, "getVMs")
	}
	for i := 0; i < len(vms); i += 1 {
		vm := vms[i].(*SVirtualMachine)
		disk, err := vm.GetIDiskById(idStr)
		if err == nil {
			return disk, nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SDatastore) fetchDisks() error {
	vms, err := self.getVMs()
	if err != nil {
		return err
	}
	allDisks := make([]cloudprovider.ICloudDisk, 0)
	for i := 0; i < len(vms); i += 1 {
		disks, err := vms[i].GetIDisks()
		if err != nil {
			return err
		}
		allDisks = append(allDisks, disks...)
	}
	self.idisks = allDisks
	return nil
}

func (self *SDatastore) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	if self.idisks != nil {
		return self.idisks, nil
	}
	err := self.fetchDisks()
	if err != nil {
		return nil, err
	}
	return self.idisks, nil
}

func (self *SDatastore) isLocalVMFS() bool {
	moStore := self.getDatastore()
	switch vmfsInfo := moStore.Info.(type) {
	case *types.VmfsDatastoreInfo:
		if vmfsInfo.Vmfs.Local == nil || *vmfsInfo.Vmfs.Local == true {
			_, err := self.getLocalHost()
			if err == nil {
				return true
			}
		}
	}
	return false
}

func (self *SDatastore) GetStorageType() string {
	moStore := self.getDatastore()
	t := strings.ToLower(moStore.Summary.Type)
	switch t {
	case "vmfs":
		if self.isLocalVMFS() {
			return api.STORAGE_LOCAL
		} else {
			return api.STORAGE_NAS
		}
	case "nfs", "nfs41":
		return api.STORAGE_NFS
	case "vsan":
		return api.STORAGE_VSAN
	case "cifs":
		return api.STORAGE_CIFS
	default:
		log.Fatalf("unsupported datastore type %s", moStore.Summary.Type)
		return ""
	}
}

func (self *SDatastore) GetMediumType() string {
	moStore := self.getDatastore()
	vmfsInfo, ok := moStore.Info.(*types.VmfsDatastoreInfo)
	if ok && vmfsInfo.Vmfs.Ssd != nil && *vmfsInfo.Vmfs.Ssd {
		return api.DISK_TYPE_SSD
	}
	return api.DISK_TYPE_ROTATE
}

func (self *SDatastore) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	conf.Add(jsonutils.NewString(self.GetName()), "name")
	conf.Add(jsonutils.NewString(self.GetGlobalId()), "id")
	conf.Add(jsonutils.NewString(self.GetDatacenterPathString()), "dc_path")
	volId, err := self.getVolumeId()
	if err != nil {
		log.Errorf("getVaolumeId fail %s", err)
	}
	conf.Add(jsonutils.NewString(volId), "volume_id")

	volType := self.getVolumeType()
	conf.Add(jsonutils.NewString(volType), "volume_type")

	volName, err := self.getVolumeName()
	if err != nil {
		log.Errorf("getVaolumeName fail %s", err)
	}
	conf.Add(jsonutils.NewString(volName), "volume_name")
	return conf
}

const dsPrefix = "ds://"

func (self *SDatastore) GetUrl() string {
	url := self.getDatastore().Info.GetDatastoreInfo().Url
	if strings.HasPrefix(url, dsPrefix) {
		url = url[len(dsPrefix):]
	}
	return url
}

func (self *SDatastore) GetMountPoint() string {
	return self.GetUrl()
}

func (self *SDatastore) HasFile(remotePath string) bool {
	dsName := fmt.Sprintf("[%s]", self.SManagedObject.GetName())
	if strings.HasPrefix(remotePath, dsName) {
		return true
	} else {
		return false
	}
}

func (self *SDatastore) cleanPath(remotePath string) string {
	dsName := fmt.Sprintf("[%s]", self.SManagedObject.GetName())
	dsUrl := self.GetUrl()
	if strings.HasPrefix(remotePath, dsName) {
		remotePath = remotePath[len(dsName):]
	} else if strings.HasPrefix(remotePath, dsUrl) {
		remotePath = remotePath[len(dsUrl):]
	}
	return strings.TrimSpace(remotePath)
}

func pathEscape(path string) string {
	segs := strings.Split(path, "/")
	for i := 0; i < len(segs); i += 1 {
		segs[i] = url.PathEscape(segs[i])
	}
	return strings.Join(segs, "/")
}

func (self *SDatastore) GetPathUrl(remotePath string) string {
	remotePath = self.cleanPath(remotePath)
	if len(remotePath) == 0 || remotePath[0] != '/' {
		remotePath = fmt.Sprintf("/%s", remotePath)
	}
	httpUrl := fmt.Sprintf("%s/folder%s", self.getManagerUri(), pathEscape(remotePath))
	params := jsonutils.NewDict()
	params.Add(jsonutils.NewString(self.SManagedObject.GetName()), "dsName")
	params.Add(jsonutils.NewString(self.GetDatacenterPathString()), "dcPath")

	return fmt.Sprintf("%s?%s", httpUrl, params.QueryString())
}

func (self *SDatastore) getPathString(path string) string {
	for len(path) > 0 && path[0] == '/' {
		path = path[1:]
	}
	return fmt.Sprintf("[%s] %s", self.SManagedObject.GetName(), path)
}

func (self *SDatastore) GetFullPath(remotePath string) string {
	remotePath = self.cleanPath(remotePath)
	return path.Join(self.GetUrl(), remotePath)
}

func (self *SDatastore) CreateIDisk(conf *cloudprovider.DiskCreateConfig) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDatastore) FileGetContent(ctx context.Context, remotePath string) ([]byte, error) {
	url := self.GetPathUrl(remotePath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	var bytes []byte

	err = self.manager.client.Do(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode == 404 {
			return cloudprovider.ErrNotFound
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("%s", resp.Status)
		}
		cont, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		bytes = cont
		return nil
	})

	return bytes, err
}

type SDatastoreFileInfo struct {
	Url      string
	Name     string
	Date     time.Time
	FileType string
	Size     uint64
}

const (
	fileListPattern = `<tr><td><a href="(?P<url>[\w\d:#@%/;$()~_?\+-=\\\.&]+)">(?P<name>[^<]+)<\/a></td><td align="right">(?P<date>[^<]+)</td><td align="right">(?P<size>[^<]+)</td></tr>`
	fileDateFormat  = "02-Jan-2006 15:04"
	fileDateFormat2 = "Mon, 2 Jan 2006 15:04:05 GMT"
)

var (
	fileListRegexp = regexp.MustCompile(fileListPattern)
)

func (self *SDatastore) ListDir(ctx context.Context, remotePath string) ([]SDatastoreFileInfo, error) {
	listContent, err := self.FileGetContent(ctx, remotePath)
	if err != nil {
		return nil, err
	}
	ret := make([]SDatastoreFileInfo, 0)
	matches := fileListRegexp.FindAllStringSubmatch(string(listContent), -1)
	for r := 0; r < len(matches); r += 1 {
		url := strings.TrimSpace(matches[r][1])
		name := strings.TrimSpace(matches[r][2])
		dateStr := strings.TrimSpace(matches[r][3])
		sizeStr := strings.TrimSpace(matches[r][4])
		var ftype string
		var size uint64
		if sizeStr == "-" {
			ftype = "dir"
			size = 0
		} else {
			ftype = "file"
			size, _ = strconv.ParseUint(sizeStr, 10, 64)
		}
		date, err := time.Parse(fileDateFormat, dateStr)
		if err != nil {
			return nil, err
		}
		info := SDatastoreFileInfo{
			Url:      url,
			Name:     name,
			FileType: ftype,
			Size:     size,
			Date:     date,
		}
		ret = append(ret, info)
	}

	return ret, nil
}

func (self *SDatastore) listPath(b *object.HostDatastoreBrowser, path string, spec types.HostDatastoreBrowserSearchSpec) ([]types.HostDatastoreBrowserSearchResults, error) {
	ctx := context.TODO()

	path = self.getDatastoreObj().Path(path)

	search := b.SearchDatastore

	task, err := search(ctx, path, &spec)
	if err != nil {
		return nil, err
	}

	info, err := task.WaitForResult(ctx, nil)
	if err != nil {
		return nil, err
	}

	switch r := info.Result.(type) {
	case types.HostDatastoreBrowserSearchResults:
		return []types.HostDatastoreBrowserSearchResults{r}, nil
	case types.ArrayOfHostDatastoreBrowserSearchResults:
		return r.HostDatastoreBrowserSearchResults, nil
	default:
		return nil, errors.Error(fmt.Sprintf("unknown result type: %T", r))
	}

}

func (self *SDatastore) ListPath(ctx context.Context, remotePath string) ([]types.HostDatastoreBrowserSearchResults, error) {
	//types.HostDatastoreBrowserSearchResults
	ds := self.getDatastoreObj()

	b, err := ds.Browser(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]types.HostDatastoreBrowserSearchResults, 0)

	spec := types.HostDatastoreBrowserSearchSpec{
		MatchPattern: []string{"*"},
		Details: &types.FileQueryFlags{
			FileType:     true,
			FileSize:     true,
			FileOwner:    types.NewBool(true), // TODO: omitempty is generated, but seems to be required
			Modification: true,
		},
	}

	for i := 0; ; i++ {
		r, err := self.listPath(b, remotePath, spec)
		if err != nil {
			// Treat the argument as a match pattern if not found as directory
			if i == 0 && types.IsFileNotFound(err) || isInvalid(err) {
				spec.MatchPattern[0] = path.Base(remotePath)
				remotePath = path.Dir(remotePath)
				continue
			}
			if types.IsFileNotFound(err) {
				return nil, errors.ErrNotFound
			}
			return nil, err
		}
		if i == 1 && len(r) == 1 && len(r[0].File) == 0 {
			return nil, errors.ErrNotFound
		}
		for n := range r {
			ret = append(ret, r[n])
		}
		break
	}
	return ret, nil
}

func isInvalid(err error) bool {
	if f, ok := err.(types.HasFault); ok {
		switch f.Fault().(type) {
		case *types.InvalidArgument:
			return true
		}
	}

	return false
}

func (self *SDatastore) CheckFile(ctx context.Context, remotePath string) (*SDatastoreFileInfo, error) {
	url := self.GetPathUrl(remotePath)

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return nil, err
	}

	var size uint64
	var date time.Time

	err = self.manager.client.Do(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode >= 400 {
			if resp.StatusCode == 404 {
				return cloudprovider.ErrNotFound
			}
			return fmt.Errorf("%s", resp.Status)
		}
		sizeStr := resp.Header.Get("Content-Length")
		size, _ = strconv.ParseUint(sizeStr, 10, 64)

		dateStr := resp.Header.Get("Date")
		date, _ = time.Parse(fileDateFormat2, dateStr)
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &SDatastoreFileInfo{Date: date, Size: size}, nil
}

func (self *SDatastore) Download(ctx context.Context, remotePath string, writer io.Writer) error {
	url := self.GetPathUrl(remotePath)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}

	err = self.manager.client.Do(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("%s", resp.Status)
		}
		buffer := make([]byte, 4096)
		for {
			rn, re := resp.Body.Read(buffer)
			if rn > 0 {
				wo := 0
				for wo < rn {
					wn, we := writer.Write(buffer[wo:rn])
					if we != nil {
						return we
					}
					wo += wn
				}
			}
			if re != nil {
				if re != io.EOF {
					return re
				} else {
					break
				}
			}
		}
		return nil
	})

	return err
}

func (self *SDatastore) Upload(ctx context.Context, remotePath string, body io.Reader) error {
	url := self.GetPathUrl(remotePath)

	req, err := http.NewRequest("PUT", url, body)
	if err != nil {
		return err
	}

	err = self.manager.client.Do(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("%s", resp.Status)
		}
		_, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// log.Debugf("upload respose %s", buffer)
		return nil
	})

	return err
}

func (self *SDatastore) FilePutContent(ctx context.Context, remotePath string, content string) error {
	return self.Upload(ctx, remotePath, strings.NewReader(content))
}

// Delete2 can delete file from this Datastore.
// isNamespace: remotePath is uuid of namespace on vsan datastore
// force: ignore nonexistent files and arguments
func (self *SDatastore) Delete2(ctx context.Context, remotePath string, isNamespace, force bool) error {
	var err error
	ds := self.getDatastoreObj()
	dc := self.datacenter.getObjectDatacenter()
	if isNamespace {
		nm := object.NewDatastoreNamespaceManager(self.manager.client.Client)
		err = nm.DeleteDirectory(ctx, dc, remotePath)
	} else {
		fm := ds.NewFileManager(dc, force)
		err = fm.Delete(ctx, remotePath)
	}

	if err != nil && types.IsFileNotFound(err) && force {
		// Ignore error
		return nil
	}
	return errors.Wrapf(err, "remove %s", remotePath)
}

func (self *SDatastore) Delete(ctx context.Context, remotePath string) error {
	url := self.GetPathUrl(remotePath)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	err = self.manager.client.Do(ctx, req, func(resp *http.Response) error {
		if resp.StatusCode >= 400 {
			return fmt.Errorf("%s", resp.Status)
		}
		_, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		// log.Debugf("delete respose %s", buffer)
		return nil
	})

	return err
}

func (self *SDatastore) DeleteVmdk(ctx context.Context, remotePath string) error {
	info, err := self.CheckFile(ctx, remotePath)
	if err != nil {
		return err
	}
	if info.Size > 4096 {
		return fmt.Errorf("not a valid vmdk file")
	}
	vmdkContent, err := self.FileGetContent(ctx, remotePath)
	if err != nil {
		return err
	}
	vmdkInfo, err := vmdkutils.Parse(string(vmdkContent))
	if err != nil {
		return err
	}
	err = self.Delete(ctx, remotePath)
	if err != nil {
		return err
	}
	if len(vmdkInfo.ExtentFile) > 0 {
		err = self.Delete(ctx, path.Join(path.Dir(remotePath), vmdkInfo.ExtentFile))
		if err != nil {
			return err
		}
	}
	return nil
}

func (self *SDatastore) GetVmdkInfo(ctx context.Context, remotePath string) (*vmdkutils.SVMDKInfo, error) {
	vmdkContent, err := self.FileGetContent(ctx, remotePath)
	if err != nil {
		return nil, err
	}
	return vmdkutils.Parse(string(vmdkContent))
}

func (self *SDatastore) CheckVmdk(ctx context.Context, remotePath string) error {
	dm := object.NewVirtualDiskManager(self.manager.client.Client)

	dc, err := self.GetDatacenter()
	if err != nil {
		return err
	}

	dcObj := dc.getObjectDatacenter()

	infoList, err := dm.QueryVirtualDiskInfo(ctx, self.getPathString(remotePath), dcObj, true)
	if err != nil {
		return err
	}

	log.Debugf("%#v", infoList)
	return nil
}

func (self *SDatastore) getDatastoreObj() *object.Datastore {
	od := object.NewDatastore(self.manager.client.Client, self.getDatastore().Self)
	od.DatacenterPath = self.GetDatacenterPathString()
	od.InventoryPath = fmt.Sprintf("%s/%s", od.DatacenterPath, self.SManagedObject.GetName())
	return od
}

func (self *SDatastore) MakeDir(remotePath string) error {
	remotePath = self.cleanPath(remotePath)

	m := object.NewFileManager(self.manager.client.Client)
	path := fmt.Sprintf("[%s] %s", self.GetRelName(), remotePath)
	return m.MakeDirectory(self.manager.context, path, self.datacenter.getObjectDatacenter(), true)
}

func (self *SDatastore) RemoveDir(ctx context.Context, remotePath string) error {
	dnm := object.NewDatastoreNamespaceManager(self.manager.client.Client)

	remotePath = self.GetFullPath(remotePath)

	dc, err := self.GetDatacenter()
	if err != nil {
		return err
	}

	dcObj := dc.getObjectDatacenter()

	return dnm.DeleteDirectory(ctx, dcObj, remotePath)
}

// CheckDirC will check that Dir 'remotePath' is exist, if not, create one.
func (self *SDatastore) CheckDirC(remotePath string) error {
	_, err := self.CheckFile(self.manager.context, remotePath)
	if err == nil {
		return nil
	}
	if errors.Cause(err) != cloudprovider.ErrNotFound {
		return err
	}
	return self.MakeDir(remotePath)

}

func (self *SDatastore) IsSysDiskStore() bool {
	return true
}

func (self *SDatastore) MoveVmdk(ctx context.Context, srcPath string, dstPath string) error {
	dm := object.NewVirtualDiskManager(self.manager.client.Client)
	defer dm.Destroy(ctx)

	srcUrl := self.GetPathUrl(srcPath)
	dstUrl := self.GetPathUrl(dstPath)
	task, err := dm.MoveVirtualDisk(ctx, srcUrl, nil, dstUrl, nil, true)
	if err != nil {
		return err
	}
	return task.Wait(ctx)
}

// domainName will abandon the rune which should't disapear
func (self *SDatastore) domainName(name string) string {
	var b bytes.Buffer
	for _, r := range name {
		if b.Len() == 0 {
			if unicode.IsLetter(r) {
				b.WriteRune(r)
			}
		} else {
			if unicode.IsDigit(r) || unicode.IsLetter(r) || r == '-' {
				b.WriteRune(r)
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// ImportVMDK will upload local vmdk 'diskFile' to the 'remotePath' of remote datastore
func (self *SDatastore) ImportVMDK(ctx context.Context, diskFile, remotePath string, host *SHost) error {
	name := fmt.Sprintf("yunioncloud.%s%d", self.domainName(remotePath)[:20], rand.Int())
	vm, err := self.ImportVM(ctx, diskFile, name, host)
	if err != nil {
		return errors.Wrap(err, "SDatastore.ImportVM")
	}

	defer func() {
		task, err := vm.Destroy(ctx)
		if err != nil {
			log.Errorf("vm.Destory: %s", err)
			return
		}

		if err = task.Wait(ctx); err != nil {
			log.Errorf("task.Wait: %s", err)
		}
	}()

	// check if 'image_cache' is esixt
	err = self.CheckDirC("image_cache")
	if err != nil {
		return errors.Wrap(err, "SDatastore.CheckDirC")
	}

	fm := self.getDatastoreObj().NewFileManager(self.datacenter.getObjectDatacenter(), true)

	// if image_cache not exist
	return fm.Move(ctx, fmt.Sprintf("[%s] %s/%s.vmdk", self.GetRelName(), name, name), fmt.Sprintf("[%s] %s",
		self.GetRelName(), remotePath))
}

func (self *SDatastore) ImportISO(ctx context.Context, isoFile, remotePath string, host *SHost) error {
	p := soap.DefaultUpload
	ds := self.getDatastoreObj()
	return ds.UploadFile(ctx, isoFile, remotePath, &p)
}

var (
	ErrInvalidFormat = errors.Error("vmdk: invalid format (must be streamOptimized)")
)

// info is used to inspect a vmdk and generate an ovf template
type info struct {
	Header struct {
		MagicNumber uint32
		Version     uint32
		Flags       uint32
		Capacity    uint64
	}

	Capacity   uint64
	Size       int64
	Name       string
	ImportName string
}

// stat looks at the vmdk header to make sure the format is streamOptimized and
// extracts the disk capacity required to properly generate the ovf descriptor.
func stat(name string) (*info, error) {
	f, err := os.Open(name)
	if err != nil {
		return nil, err
	}

	var (
		di  info
		buf bytes.Buffer
	)

	_, err = io.CopyN(&buf, f, int64(binary.Size(di.Header)))
	fi, _ := f.Stat()
	_ = f.Close()
	if err != nil {
		return nil, err
	}

	err = binary.Read(&buf, binary.LittleEndian, &di.Header)
	if err != nil {
		return nil, err
	}

	if di.Header.MagicNumber != 0x564d444b { // SPARSE_MAGICNUMBER
		return nil, ErrInvalidFormat
	}

	if di.Header.Flags&(1<<16) == 0 { // SPARSEFLAG_COMPRESSED
		// Needs to be converted, for example:
		//   vmware-vdiskmanager -r src.vmdk -t 5 dst.vmdk
		//   qemu-img convert -O vmdk -o subformat=streamOptimized src.vmdk dst.vmdk
		return nil, ErrInvalidFormat
	}

	di.Capacity = di.Header.Capacity * 512 // VMDK_SECTOR_SIZE
	di.Size = fi.Size()
	di.Name = filepath.Base(name)
	di.ImportName = strings.TrimSuffix(di.Name, ".vmdk")

	return &di, nil
}

// ovf returns an expanded descriptor template
func (di *info) ovf() (string, error) {
	var buf bytes.Buffer

	tmpl, err := template.New("ovf").Parse(ovfTemplate)
	if err != nil {
		return "", errors.Wrapf(err, "template.New")
	}

	err = tmpl.Execute(&buf, di)
	if err != nil {
		return "", errors.Wrapf(err, "tmpl.Execute")
	}

	return buf.String(), nil
}

// ImportParams contains the set of optional params to the Import function.
// Note that "optional" may depend on environment, such as ESX or vCenter.
type ImportParams struct {
	Path       string
	Logger     progress.Sinker
	Type       types.VirtualDiskType
	Force      bool
	Datacenter *object.Datacenter
	Pool       *object.ResourcePool
	Folder     *object.Folder
	Host       *object.HostSystem
}

// ImportVM will import a vm by uploading a local vmdk
func (self *SDatastore) ImportVM(ctx context.Context, diskFile, name string, host *SHost) (*object.VirtualMachine, error) {

	var (
		c         = self.manager.client.Client
		datastore = self.getDatastoreObj()
	)

	m := ovf.NewManager(c)

	disk, err := stat(diskFile)
	if err != nil {
		return nil, errors.Wrap(err, "stat")
	}

	disk.ImportName = name

	// Expand the ovf template
	descriptor, err := disk.ovf()
	if err != nil {
		return nil, errors.Wrap(err, "disk.ovf")
	}

	folders, err := self.datacenter.getObjectDatacenter().Folders(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "Folders")
	}
	pool, err := host.GetResourcePool()
	if err != nil {
		return nil, errors.Wrap(err, "getResourcePool")
	}

	kind := types.VirtualDiskTypeThin

	params := types.OvfCreateImportSpecParams{
		DiskProvisioning: string(kind),
		EntityName:       disk.ImportName,
	}

	spec, err := m.CreateImportSpec(ctx, descriptor, pool, datastore, params)
	if err != nil {
		return nil, err
	}
	if spec.Error != nil {
		return nil, errors.Error(spec.Error[0].LocalizedMessage)
	}

	lease, err := pool.ImportVApp(ctx, spec.ImportSpec, folders.VmFolder, host.GetHostSystem())
	if err != nil {
		return nil, err
	}

	info, err := lease.Wait(ctx, spec.FileItem)
	if err != nil {
		return nil, err
	}

	f, err := os.Open(diskFile)
	if err != nil {
		return nil, err
	}

	lr := newLeaseLogger("upload vmdk", 5)

	lr.Log()
	defer lr.End()

	opts := soap.Upload{
		ContentLength: disk.Size,
		Progress:      lr,
	}

	u := lease.StartUpdater(ctx, info)
	defer u.Done()

	item := info.Items[0] // we only have 1 disk to upload

	err = lease.Upload(ctx, item, f, opts)

	_ = f.Close()

	if err != nil {
		return nil, errors.Wrap(err, "lease.Upload")
	}

	if err = lease.Complete(ctx); err != nil {
		log.Debugf("lease complete error: %s, sleep 1s and try again", err)
		time.Sleep(time.Second)
		if err = lease.Complete(ctx); err != nil {
			return nil, errors.Wrap(err, "lease.Complete")
		}
	}

	return object.NewVirtualMachine(c, info.Entity), nil
}
