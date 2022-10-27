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

package cephutils

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type CephClient struct {
	monHost  string
	key      string
	pool     string
	cephConf string
	keyConf  string
}

func (self *CephClient) Close() error {
	if len(self.keyConf) > 0 {
		os.Remove(self.keyConf)
	}
	return os.Remove(self.cephConf)
}

func (self *CephClient) SetPool(pool string) {
	self.pool = pool
}

type cephStats struct {
	Stats struct {
		TotalBytes        int64   `json:"total_bytes"`
		TotalAvailBytes   int64   `json:"total_avail_bytes"`
		TotalUsedBytes    int64   `json:"total_used_bytes"`
		TotalUsedRawBytes int64   `json:"total_used_raw_bytes"`
		TotalUsedRawRatio float64 `json:"total_used_raw_ratio"`
		NumOsds           int     `json:"num_osds"`
		NumPerPoolOsds    int     `json:"num_per_pool_osds"`
	} `json:"stats"`
	StatsByClass struct {
		Hdd struct {
			TotalBytes        int64   `json:"total_bytes"`
			TotalAvailBytes   int64   `json:"total_avail_bytes"`
			TotalUsedBytes    int64   `json:"total_used_bytes"`
			TotalUsedRawBytes int64   `json:"total_used_raw_bytes"`
			TotalUsedRawRatio float64 `json:"total_used_raw_ratio"`
		} `json:"hdd"`
	} `json:"stats_by_class"`
	Pools []struct {
		Name  string `json:"name"`
		ID    int    `json:"id"`
		Stats struct {
			Stored      int   `json:"stored"`
			Objects     int   `json:"objects"`
			KbUsed      int   `json:"kb_used"`
			BytesUsed   int   `json:"bytes_used"`
			PercentUsed int   `json:"percent_used"`
			MaxAvail    int64 `json:"max_avail"`
		} `json:"stats"`
	} `json:"pools"`
}

type SCapacity struct {
	CapacitySizeKb     int64
	UsedCapacitySizeKb int64
}

func (self *CephClient) Output(name string, opts []string) (jsonutils.JSONObject, error) {
	return self.output(name, opts)
}

func (self *CephClient) output(name string, opts []string) (jsonutils.JSONObject, error) {
	opts = append([]string{"--format", "json"}, opts...)
	proc := procutils.NewRemoteCommandAsFarAsPossible(name, opts...)
	outb, err := proc.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stdout pipe")
	}
	defer outb.Close()

	errb, err := proc.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stderr pipe")
	}
	defer errb.Close()

	if err := proc.Start(); err != nil {
		return nil, errors.Wrap(err, "start ceph process")
	}

	stdoutPut, err := ioutil.ReadAll(outb)
	if err != nil {
		return nil, err
	}
	stderrPut, err := ioutil.ReadAll(errb)
	if err != nil {
		return nil, err
	}

	if err := proc.Wait(); err != nil {
		return nil, errors.Wrapf(err, "stderr %q", stderrPut)
	}
	return jsonutils.Parse(stdoutPut)
}

func (self *CephClient) run(name string, opts []string) error {
	output, err := procutils.NewRemoteCommandAsFarAsPossible(name, opts...).Output()
	if err != nil {
		return errors.Wrapf(err, "%s %s", name, string(output))
	}
	return nil
}

func (self *CephClient) options() []string {
	opts := []string{"--conf", self.cephConf}
	if len(self.keyConf) > 0 {
		opts = append(opts, []string{"--keyring", self.keyConf}...)
	}
	return opts
}

func (self *CephClient) CreateImage(name string, sizeMb int64) (*SImage, error) {
	opts := self.options()
	image := &SImage{name: name, client: self}
	opts = append(opts, []string{"create", image.GetName(), "--size", fmt.Sprintf("%dM", sizeMb)}...)
	return image, self.run("rbd", opts)
}

func (self *CephClient) GetCapacity() (*SCapacity, error) {
	result := &SCapacity{}
	opts := self.options()
	opts = append(opts, "df")
	resp, err := self.output("ceph", opts)
	if err != nil {
		return nil, errors.Wrapf(err, "output")
	}
	stats := cephStats{}
	err = resp.Unmarshal(&stats)
	if err != nil {
		return nil, errors.Wrapf(err, "ret.Unmarshal")
	}
	result.CapacitySizeKb = stats.Stats.TotalBytes / 1024
	result.UsedCapacitySizeKb = stats.Stats.TotalUsedBytes / 1024
	for _, pool := range stats.Pools {
		if pool.Name == self.pool {
			result.UsedCapacitySizeKb = int64(pool.Stats.KbUsed)
			if pool.Stats.MaxAvail > 0 {
				result.CapacitySizeKb = int64(pool.Stats.MaxAvail / 1024)
			}
		}
	}
	if result.CapacitySizeKb == 0 {
		log.Warningf("cluster size is zero, output is: %s", resp)
	}
	return result, nil
}

func writeFile(pattern string, content string) (string, error) {
	file, err := ioutil.TempFile("", pattern)
	if err != nil {
		return "", errors.Wrapf(err, "TempFile")
	}
	defer file.Close()
	name := file.Name()
	_, err = file.Write([]byte(content))
	if err != nil {
		return name, errors.Wrapf(err, "write")
	}
	return name, nil
}

func (cli *CephClient) ShowConf() error {
	conf, err := fileutils2.FileGetContents(cli.cephConf)
	if err != nil {
		return errors.Errorf("fail to open conf file")
	}
	key, err := fileutils2.FileGetContents(cli.keyConf)
	if err != nil {
		return errors.Errorf("fail to open key file")
	}
	fmt.Println("ceph.conf")
	fmt.Println(conf)
	fmt.Println("key.conf")
	fmt.Println(key)
	return nil
}

func NewClient(monHost, key, pool string) (*CephClient, error) {
	client := &CephClient{
		monHost: monHost,
		key:     key,
		pool:    pool,
	}
	var err error
	if len(client.key) > 0 {
		keyring := fmt.Sprintf(`[client.admin]
	key = %s
`, client.key)
		client.keyConf, err = writeFile("ceph.*.keyring", keyring)
		if err != nil {
			return nil, errors.Wrapf(err, "write keyring")
		}
	}
	monHosts := []string{}
	for _, monHost := range strings.Split(client.monHost, ",") {
		monHosts = append(monHosts, fmt.Sprintf(`[%s]`, monHost))
	}
	conf := fmt.Sprintf(`[global]
mon host = %s
rados mon op timeout = 5
rados osd_op timeout = 1200
client mount timeout = 120
`, strings.Join(monHosts, ","))
	if len(client.key) == 0 {
		conf = fmt.Sprintf(`%s
auth_cluster_required = none
auth_service_required = none
auth_client_required = none
`, conf)
	} else {
		conf = fmt.Sprintf(`%s
keyring = %s
`, conf, client.keyConf)
	}
	client.cephConf, err = writeFile("ceph.*.conf", conf)
	if err != nil {
		return nil, errors.Wrapf(err, "write file")
	}
	return client, nil
}

type SImage struct {
	name   string
	client *CephClient
}

func (self *SImage) GetName() string {
	return fmt.Sprintf("%s/%s", self.client.pool, self.name)
}

func (self *CephClient) ListImages() ([]string, error) {
	result := []string{}
	opts := self.options()
	opts = append(opts, []string{"ls", self.pool}...)
	resp, err := self.output("rbd", opts)
	if err != nil {
		return nil, err
	}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, errors.Wrapf(err, "ret.Unmarshal")
	}
	return result, nil
}

func (self *CephClient) GetImage(name string) (*SImage, error) {
	images, err := self.ListImages()
	if err != nil {
		return nil, errors.Wrapf(err, "ListImages")
	}
	if !utils.IsInStringArray(name, images) {
		return nil, cloudprovider.ErrNotFound
	}
	return &SImage{name: name, client: self}, nil
}

type SImageInfo struct {
	Name            string        `json:"name"`
	ID              string        `json:"id"`
	SizeByte        int64         `json:"size"`
	Objects         int           `json:"objects"`
	Order           int           `json:"order"`
	ObjectSize      int           `json:"object_size"`
	SnapshotCount   int           `json:"snapshot_count"`
	BlockNamePrefix string        `json:"block_name_prefix"`
	Format          int           `json:"format"`
	Features        []string      `json:"features"`
	OpFeatures      []interface{} `json:"op_features"`
	Flags           []interface{} `json:"flags"`
	CreateTimestamp string        `json:"create_timestamp"`
	AccessTimestamp string        `json:"access_timestamp"`
	ModifyTimestamp string        `json:"modify_timestamp"`
}

func (self *SImage) options() []string {
	return self.client.options()
}

func (self *SImage) GetInfo() (*SImageInfo, error) {
	opts := self.options()
	opts = append(opts, []string{"info", self.GetName()}...)
	resp, err := self.client.output("rbd", opts)
	if err != nil {
		return nil, err
	}
	info := &SImageInfo{}
	return info, resp.Unmarshal(info)
}

func (self *SImage) ListSnapshots() ([]SSnapshot, error) {
	opts := self.options()
	opts = append(opts, []string{"snap", "ls", self.GetName()}...)
	resp, err := self.client.output("rbd", opts)
	if err != nil {
		return nil, err
	}
	result := []SSnapshot{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, errors.Wrapf(err, "ret.Unmarshal")
	}
	for i := range result {
		result[i].image = self
	}
	return result, nil
}

func (self *SImage) GetSnapshot(name string) (*SSnapshot, error) {
	snaps, err := self.ListSnapshots()
	if err != nil {
		return nil, errors.Wrapf(err, "ListSnapshots")
	}
	for i := range snaps {
		if snaps[i].Name == name {
			return &snaps[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SImage) IsSnapshotExist(name string) (bool, error) {
	_, err := self.GetSnapshot(name)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false, nil
		}
		return false, errors.Wrapf(err, "GetSnapshot")
	}
	return true, nil
}

type SSnapshot struct {
	Name      string
	Id        string
	Size      int64
	Protected bool
	Timestamp string

	image *SImage
}

func (self *SSnapshot) Rollback() error {
	opts := self.options()
	opts = append(opts, []string{"snap", "rollback", self.GetName()}...)
	return self.image.client.run("rbd", opts)
}

func (self *SSnapshot) options() []string {
	return self.image.options()
}

func (self *SSnapshot) GetName() string {
	return fmt.Sprintf("%s@%s", self.image.GetName(), self.Name)
}

func (self *SSnapshot) Unprotect() error {
	opts := self.options()
	opts = append(opts, []string{"snap", "unprotect", self.GetName()}...)
	err := self.image.client.run("rbd", opts)
	if err != nil {
		if strings.Contains(err.Error(), "snap is already unprotected") {
			return nil
		}
		return errors.Wrapf(err, "Unprotect")
	}
	self.Protected = false
	return nil
}

func (self *SSnapshot) Protect() error {
	if self.Protected {
		return nil
	}
	opts := self.options()
	opts = append(opts, []string{"snap", "protect", self.GetName()}...)
	err := self.image.client.run("rbd", opts)
	if err == nil {
		self.Protected = true
	}
	return err
}

func (self *SSnapshot) Remove() error {
	opts := self.options()
	opts = append(opts, []string{"snap", "rm", self.GetName()}...)
	return self.image.client.run("rbd", opts)
}

func (self *SSnapshot) Delete() error {
	pool := self.image.client.pool
	defer self.image.client.SetPool(pool)

	children, err := self.ListChildren()
	if err != nil {
		return errors.Wrapf(err, "ListChildren")
	}

	for i := range children {
		self.image.client.SetPool(children[i].Pool)
		image, err := self.image.client.GetImage(children[i].Image)
		if err != nil {
			return errors.Wrapf(err, "GetImage(%s/%s)", children[i].Pool, children[i].Image)
		}
		err = image.Flatten()
		if err != nil {
			return errors.Wrapf(err, "Flatten")
		}
	}

	err = self.Unprotect()
	if err != nil {
		log.Errorf("Unprotect %s failed: %s", self.GetName(), err)
	}

	return self.Remove()
}

type SChildren struct {
	Pool          string
	PoolNamespace string
	Image         string
}

func (self *SSnapshot) ListChildren() ([]SChildren, error) {
	opts := self.options()
	opts = append(opts, []string{"children", self.GetName()}...)
	resp, err := self.image.client.output("rbd", opts)
	if err != nil {
		return nil, errors.Wrapf(err, "ListChildren")
	}
	chidren := []SChildren{}
	return chidren, resp.Unmarshal(&chidren)
}

func (self *SImage) Resize(sizeMb int64) error {
	opts := self.options()
	opts = append(opts, []string{"resize", self.GetName(), "--size", fmt.Sprintf("%dM", sizeMb)}...)
	return self.client.run("rbd", opts)
}

func (self *SImage) Remove() error {
	opts := self.options()
	opts = append(opts, []string{"rm", self.GetName()}...)
	return self.client.run("rbd", opts)
}

func (self *SImage) Flatten() error {
	opts := self.options()
	opts = append(opts, []string{"flatten", self.GetName()}...)
	return self.client.run("rbd", opts)
}

func (self *SImage) Delete() error {
	snapshots, err := self.ListSnapshots()
	if err != nil {
		return errors.Wrapf(err, "ListSnapshots")
	}
	for i := range snapshots {
		err := snapshots[i].Delete()
		if err != nil {
			return errors.Wrapf(err, "delete snapshot %s", snapshots[i].GetName())
		}
	}
	return self.Remove()
}

func (self *SImage) Rename(name string) error {
	opts := self.options()
	opts = append(opts, []string{"rename", self.GetName(), fmt.Sprintf("%s/%s", self.client.pool, name)}...)
	return self.client.run("rbd", opts)
}

func (self *SImage) CreateSnapshot(name string) (*SSnapshot, error) {
	snap := &SSnapshot{Name: name, image: self}
	opts := self.options()
	opts = append(opts, []string{"snap", "create", snap.GetName()}...)
	if err := self.client.run("rbd", opts); err != nil {
		return nil, errors.Wrap(err, "snap create")
	}
	if err := snap.Protect(); err != nil {
		log.Errorf("failed protect snap %s: %s", snap.GetName(), err)
	}
	return snap, nil
}

func (self *SImage) Clone(ctx context.Context, pool, name string) error {
	lockman.LockRawObject(ctx, "rbd_image_cache", self.GetName())
	defer lockman.ReleaseRawObject(ctx, "rbd_image_cache", self.GetName())

	var findOrCreateSnap = func() (*SSnapshot, error) {
		snaps, err := self.ListSnapshots()
		if err != nil {
			return nil, errors.Wrapf(err, "ListSnapshots")
		}
		for i := range snaps {
			if snaps[i].Name == name {
				return &snaps[i], nil
			}
		}
		snap, err := self.CreateSnapshot(name)
		if err != nil {
			return nil, errors.Wrapf(err, "CreateSnapshot")
		}
		return snap, nil
	}
	snap, err := findOrCreateSnap()
	if err != nil {
		return errors.Wrapf(err, "findOrCreateSnap")
	}
	err = snap.Clone(pool, name)
	if err != nil {
		return errors.Wrapf(err, "clone %s/%s", pool, name)
	}

	_pool := self.client.pool

	// use current pool
	self.client.SetPool(pool)
	// recover previous pool
	defer self.client.SetPool(_pool)

	img, err := self.client.GetImage(name)
	if err != nil {
		return errors.Wrapf(err, "GetImage(%s) after clone", name)
	}
	return img.Flatten()
}

func (self *SSnapshot) Clone(pool, name string) error {
	err := self.Protect()
	if err != nil {
		log.Warningf("protect %s error: %v", self.GetName(), err)
	}
	opts := self.options()
	opts = append(opts, []string{"clone", self.GetName(), fmt.Sprintf("%s/%s", pool, name)}...)
	return self.image.client.run("rbd", opts)
}
