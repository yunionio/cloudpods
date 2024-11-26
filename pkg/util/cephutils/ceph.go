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
	"time"

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

	timeout int
}

func (cli *CephClient) Close() error {
	if len(cli.keyConf) > 0 {
		os.Remove(cli.keyConf)
	}
	return os.Remove(cli.cephConf)
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

func (cli *CephClient) Output(name string, opts []string) (jsonutils.JSONObject, error) {
	return cli.output(name, opts, false)
}

func (cli *CephClient) output(name string, opts []string, timeout bool) (jsonutils.JSONObject, error) {
	cmds := []string{name, "--format", "json"}
	cmds = append(cmds, opts...)
	if timeout {
		cmds = append([]string{"timeout", "--signal=KILL", fmt.Sprintf("%ds", cli.timeout)}, cmds...)
	}
	proc := procutils.NewRemoteCommandAsFarAsPossible(cmds[0], cmds[1:]...)
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

func (cli *CephClient) run(name string, opts []string, timeout bool) error {
	cmds := append([]string{name}, opts...)
	if timeout {
		cmds = append([]string{"timeout", "--signal=KILL", fmt.Sprintf("%ds", cli.timeout)}, cmds...)
	}
	output, err := procutils.NewRemoteCommandAsFarAsPossible(cmds[0], cmds[1:]...).Output()
	if err != nil {
		return errors.Wrapf(err, "%s %s", name, string(output))
	}
	return nil
}

func (cli *CephClient) options() []string {
	opts := []string{"--conf", cli.cephConf}
	if len(cli.keyConf) > 0 {
		opts = append(opts, []string{"--keyring", cli.keyConf}...)
	}
	return opts
}

func (cli *CephClient) CreateImage(name string, sizeMb int64) (*SImage, error) {
	opts := cli.options()
	image := &SImage{name: name, client: cli}
	opts = append(opts, []string{"create", image.GetName(), "--size", fmt.Sprintf("%dM", sizeMb)}...)
	return image, cli.run("rbd", opts, false)
}

/*
 * {"kb_used":193408,"bytes_used":198049792,"percent_used":0.32,"bytes_used2":0,"percent_used2":0.00,"osd_max_used":0,"osd_max_used_ratio":0.32,"max_avail":61003137024,"objects":1,"origin_bytes":0,"compress_bytes":0}
 * {"stored":6198990973173,"objects":1734699,"kb_used":12132844593,"bytes_used":12424032862699,"percent_used":0.30800202488899231,"max_avail":13956734255104}
 */
func (cli *CephClient) GetCapacity() (*SCapacity, error) {
	result := &SCapacity{}
	opts := cli.options()
	opts = append(opts, "df")
	resp, err := cli.output("ceph", opts, true)
	if err != nil {
		return nil, errors.Wrapf(err, "output")
	}
	stats := cephStats{}
	err = resp.Unmarshal(&stats)
	if err != nil {
		return nil, errors.Wrapf(err, "ret.Unmarshal %s", resp)
	}
	result.CapacitySizeKb = stats.Stats.TotalBytes / 1024
	result.UsedCapacitySizeKb = stats.Stats.TotalUsedBytes / 1024
	for _, pool := range stats.Pools {
		if pool.Name == cli.pool {
			if pool.Stats.Stored > 0 {
				result.UsedCapacitySizeKb = int64(pool.Stats.Stored / 1024)
			} else {
				result.UsedCapacitySizeKb = int64(pool.Stats.BytesUsed / 1024)
			}
			if pool.Stats.MaxAvail > 0 {
				result.CapacitySizeKb = int64(pool.Stats.MaxAvail/1024) + result.UsedCapacitySizeKb
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

func (cli *CephClient) SetTimeout(timeout int) {
	cli.timeout = timeout
}

const DEFAULT_TIMTOUT_SECOND = 15

func NewClient(monHost, key, pool string, enableMessengerV2 bool) (*CephClient, error) {
	client := &CephClient{
		monHost: monHost,
		key:     key,
		pool:    pool,
		timeout: DEFAULT_TIMTOUT_SECOND,
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
	if enableMessengerV2 {
		for _, monHost := range strings.Split(client.monHost, ",") {
			monHosts = append(monHosts, fmt.Sprintf(`[v2:%s:3300/0,v1:%s:6789/0]`, monHost, monHost))
		}
	} else {
		for _, monHost := range strings.Split(client.monHost, ",") {
			monHosts = append(monHosts, fmt.Sprintf(`[%s]`, monHost))
		}
	}
	client.monHost = strings.Join(monHosts, ",")

	conf := fmt.Sprintf(`[global]
mon host = %s
rados mon op timeout = 5
rados osd_op timeout = 1200
client mount timeout = 120
`, client.monHost)
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

func (cli *CephClient) Child(pool string) *CephClient {
	newCli := *cli
	newCli.pool = pool
	return &newCli
}

type SImage struct {
	name   string
	client *CephClient
}

func (img *SImage) GetName() string {
	return fmt.Sprintf("%s/%s", img.client.pool, img.name)
}

func (cli *CephClient) ListImages() ([]string, error) {
	result := []string{}
	opts := cli.options()
	opts = append(opts, []string{"ls", cli.pool}...)
	resp, err := cli.output("rbd", opts, true)
	if err != nil {
		return nil, err
	}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, errors.Wrapf(err, "ret.Unmarshal")
	}
	return result, nil
}

func (cli *CephClient) GetImage(name string) (*SImage, error) {
	images, err := cli.ListImages()
	if err != nil {
		return nil, errors.Wrapf(err, "ListImages")
	}
	if !utils.IsInStringArray(name, images) {
		return nil, cloudprovider.ErrNotFound
	}
	return &SImage{name: name, client: cli}, nil
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
	CreateTimestamp time.Time     `json:"create_timestamp"`
	AccessTimestamp time.Time     `json:"access_timestamp"`
	ModifyTimestamp time.Time     `json:"modify_timestamp"`
}

func (img *SImage) options() []string {
	return img.client.options()
}

func (img *SImage) GetInfo() (*SImageInfo, error) {
	opts := img.options()
	opts = append(opts, []string{"info", img.GetName()}...)
	resp, err := img.client.output("rbd", opts, true)
	if err != nil {
		return nil, err
	}
	info := &SImageInfo{}
	return info, resp.Unmarshal(info)
}

func (img *SImage) ListSnapshots() ([]SSnapshot, error) {
	opts := img.options()
	opts = append(opts, []string{"snap", "ls", img.GetName()}...)
	resp, err := img.client.output("rbd", opts, true)
	if err != nil {
		return nil, err
	}
	result := []SSnapshot{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, errors.Wrapf(err, "ret.Unmarshal")
	}
	for i := range result {
		result[i].image = img
	}
	return result, nil
}

func (img *SImage) GetSnapshot(name string) (*SSnapshot, error) {
	snaps, err := img.ListSnapshots()
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

func (img *SImage) IsSnapshotExist(name string) (bool, error) {
	_, err := img.GetSnapshot(name)
	if err != nil {
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false, nil
		}
		return false, errors.Wrapf(err, "GetSnapshot")
	}
	return true, nil
}

type SSnapshot struct {
	Name string
	Id   string
	Size int64
	// Protected bool
	Timestamp string

	image *SImage
}

func (snap *SSnapshot) Rollback() error {
	opts := snap.options()
	opts = append(opts, []string{"snap", "rollback", snap.GetName()}...)
	return snap.image.client.run("rbd", opts, false)
}

func (snap *SSnapshot) options() []string {
	return snap.image.options()
}

func (snap *SSnapshot) GetName() string {
	return fmt.Sprintf("%s@%s", snap.image.GetName(), snap.Name)
}

func (snap *SSnapshot) Unprotect() error {
	opts := snap.options()
	opts = append(opts, []string{"snap", "unprotect", snap.GetName()}...)
	err := snap.image.client.run("rbd", opts, true)
	if err != nil {
		if strings.Contains(err.Error(), "snap is already unprotected") {
			// snap.Protected = false
			return nil
		}
		return errors.Wrapf(err, "unprotect")
	}
	// snap.Protected = false
	return nil
}

func (snap *SSnapshot) protect() error {
	// if snap.Protected {
	//	return nil
	// }
	opts := snap.options()
	opts = append(opts, []string{"snap", "protect", snap.GetName()}...)
	err := snap.image.client.run("rbd", opts, true)
	if err != nil {
		if strings.Contains(err.Error(), "snap is already protected") {
			return nil
		}
		return errors.Wrap(err, "protect")
	}
	// if err == nil {
	//	snap.Protected = true
	// }
	return err
}

func (snap *SSnapshot) Remove() error {
	opts := snap.options()
	opts = append(opts, []string{"snap", "rm", snap.GetName()}...)
	return snap.image.client.run("rbd", opts, false)
}

func (snap *SSnapshot) Delete() error {
	children, err := snap.ListChildren()
	if err != nil {
		return errors.Wrapf(err, "ListChildren")
	}

	for i := range children {
		tmpCli := snap.image.client.Child(children[i].Pool)
		image, err := tmpCli.GetImage(children[i].Image)
		if err != nil {
			return errors.Wrapf(err, "GetImage(%s/%s)", children[i].Pool, children[i].Image)
		}
		err = image.Flatten()
		if err != nil {
			return errors.Wrapf(err, "Flatten")
		}
	}

	// always try to unprotect
	err = snap.Unprotect()
	if err != nil {
		log.Errorf("Unprotect %s failed: %s", snap.GetName(), err)
	}

	return snap.Remove()
}

type SChildren struct {
	Pool          string
	PoolNamespace string
	Image         string
}

func (snap *SSnapshot) ListChildren() ([]SChildren, error) {
	opts := snap.options()
	opts = append(opts, []string{"children", snap.GetName()}...)
	resp, err := snap.image.client.output("rbd", opts, true)
	if err != nil {
		return nil, errors.Wrapf(err, "ListChildren")
	}
	chidren := []SChildren{}
	return chidren, resp.Unmarshal(&chidren)
}

func (img *SImage) Resize(sizeMb int64) error {
	opts := img.options()
	opts = append(opts, []string{"resize", img.GetName(), "--size", fmt.Sprintf("%dM", sizeMb)}...)
	return img.client.run("rbd", opts, false)
}

func (img *SImage) Remove() error {
	opts := img.options()
	opts = append(opts, []string{"rm", img.GetName()}...)
	return img.client.run("rbd", opts, false)
}

func (img *SImage) Flatten() error {
	opts := img.options()
	opts = append(opts, []string{"flatten", img.GetName()}...)
	return img.client.run("rbd", opts, false)
}

func (img *SImage) Delete() error {
	snapshots, err := img.ListSnapshots()
	if err != nil {
		return errors.Wrapf(err, "ListSnapshots")
	}
	for i := range snapshots {
		err := snapshots[i].Delete()
		if err != nil {
			return errors.Wrapf(err, "delete snapshot %s", snapshots[i].GetName())
		}
	}
	return img.Remove()
}

func (img *SImage) Rename(name string) error {
	opts := img.options()
	opts = append(opts, []string{"rename", img.GetName(), fmt.Sprintf("%s/%s", img.client.pool, name)}...)
	return img.client.run("rbd", opts, false)
}

func (img *SImage) CreateSnapshot(name string) (*SSnapshot, error) {
	snap := &SSnapshot{Name: name, image: img}
	opts := img.options()
	opts = append(opts, []string{"snap", "create", snap.GetName()}...)
	if err := img.client.run("rbd", opts, false); err != nil {
		return nil, errors.Wrap(err, "snap create")
	}
	return snap, nil
}

func (img *SImage) Clone(ctx context.Context, pool, name string) (*SImage, error) {
	lockman.LockRawObject(ctx, "rbd_image_cache", img.GetName())
	defer lockman.ReleaseRawObject(ctx, "rbd_image_cache", img.GetName())

	tmpSnapName := "snap-" + utils.GenRequestId(12)
	tmpSnap, err := img.CreateSnapshot(tmpSnapName)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSnapshot")
	}
	defer tmpSnap.Delete()

	newimg, err := tmpSnap.Clone(pool, name, true)
	if err != nil {
		return nil, errors.Wrapf(err, "clone %s/%s", pool, name)
	}

	return newimg, nil
}

func (snap *SSnapshot) Clone(pool, name string, flattern bool) (*SImage, error) {
	err := snap.protect()
	if err != nil {
		log.Warningf("protect %s error: %v", snap.GetName(), err)
		return nil, errors.Wrap(err, "Protect")
	}
	if flattern {
		defer snap.Unprotect()
	}

	opts := snap.options()
	opts = append(opts, []string{"clone", snap.GetName(), fmt.Sprintf("%s/%s", pool, name)}...)
	err = snap.image.client.run("rbd", opts, false)
	if err != nil {
		return nil, errors.Wrap(err, "clone")
	}
	newimg, err := snap.image.client.Child(pool).GetImage(name)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImage(%s) after clone", name)
	}
	if flattern {
		err = newimg.Flatten()
		if err != nil {
			return nil, errors.Wrap(err, "flattern")
		}
	}
	return newimg, nil
}
