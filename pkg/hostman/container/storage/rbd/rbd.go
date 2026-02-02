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

package rbd

import (
	"fmt"
	"path/filepath"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/container/storage"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	storage.RegisterDriver(newRbd())
}

type rbd struct{}

func newRbd() *rbd {
	return &rbd{}
}

func (r rbd) GetType() storage.StorageType {
	return storage.STORAGE_TYPE_RBD
}

func parseRbdPath(diskPath string) (pool, image, confPath, keyringPath string, err error) {
	if !strings.HasPrefix(diskPath, "rbd:") {
		return "", "", "", "", errors.Errorf("invalid rbd path: %s", diskPath)
	}
	diskPath = strings.TrimPrefix(diskPath, "rbd:")
	parts := strings.SplitN(diskPath, ":", 2)
	if len(parts) < 1 || parts[0] == "" {
		return "", "", "", "", errors.Errorf("invalid rbd path: missing pool/image")
	}
	poolImage := parts[0]
	slash := strings.Index(poolImage, "/")
	if slash <= 0 {
		return "", "", "", "", errors.Errorf("invalid rbd path: missing pool/image in %s", poolImage)
	}
	pool = poolImage[:slash]
	image = poolImage[slash+1:]
	if pool == "" || image == "" {
		return "", "", "", "", errors.Errorf("invalid rbd path: empty pool or image")
	}
	confPath = ""
	if len(parts) == 2 && parts[1] != "" {
		confPrefix := "conf="
		if strings.HasPrefix(parts[1], confPrefix) {
			confPath = strings.TrimPrefix(parts[1], confPrefix)
		}
	}
	if confPath == "" {
		return "", "", "", "", errors.Errorf("invalid rbd path: missing conf= in %s", diskPath)
	}
	keyringPath = filepath.Join(filepath.Dir(confPath), "ceph.keyring")
	return pool, image, confPath, keyringPath, nil
}

func imageSpec(pool, image string) string {
	return fmt.Sprintf("%s/%s", pool, image)
}

type rbdDeviceInfo struct {
	Id        int    `json:"id"`
	Pool      string `json:"pool"`
	Namespace string `json:"namespace"`
	Image     string `json:"image"`
	Name      string `json:"name"`
	Snap      string `json:"snap"`
	Device    string `json:"device"`
}

func (r rbd) listMappedDevices(confPath, keyringPath string) (map[string]string, error) {
	args := []string{"device", "list", "--format", "json"}
	if confPath != "" {
		args = append(args, "--conf", confPath)
	}
	if keyringPath != "" && fileutils2.Exists(keyringPath) {
		args = append(args, "--keyring", keyringPath)
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("rbd", args...).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "rbd device list: %s", string(out))
	}
	jsonObj, err := jsonutils.Parse(out)
	if err != nil {
		return nil, errors.Wrapf(err, "parse rbd device list json output: %s", string(out))
	}
	devices, err := jsonObj.GetArray()
	if err != nil {
		return nil, errors.Wrapf(err, "get devices array from json: %s", string(out))
	}
	result := make(map[string]string)
	for _, devObj := range devices {
		devInfo := rbdDeviceInfo{}
		if err := devObj.Unmarshal(&devInfo); err != nil {
			log.Warningf("failed to unmarshal device info: %v, skip", err)
			continue
		}
		image := devInfo.Image
		if image == "" {
			image = devInfo.Name
		}
		if devInfo.Pool == "" || image == "" || devInfo.Device == "" {
			continue
		}
		spec := imageSpec(devInfo.Pool, image)
		result[spec] = devInfo.Device
	}
	return result, nil
}

func (r rbd) CheckConnect(diskPath string) (string, bool, error) {
	pool, image, confPath, keyringPath, err := parseRbdPath(diskPath)
	if err != nil {
		return "", false, err
	}
	spec := imageSpec(pool, image)
	mapped, err := r.listMappedDevices(confPath, keyringPath)
	if err != nil {
		return "", false, err
	}
	dev, ok := mapped[spec]
	if !ok {
		return "", false, nil
	}
	devPath := r.checkPartition(dev)
	return devPath, true, nil
}

func (r rbd) checkPartition(devName string) string {
	// /dev/rbd0 -> /dev/rbd0p1
	partPath := devName + "p1"
	if fileutils2.Exists(partPath) {
		return partPath
	}
	return devName
}

func (r rbd) ConnectDisk(diskPath string) (string, error) {
	pool, image, confPath, keyringPath, err := parseRbdPath(diskPath)
	if err != nil {
		return "", err
	}
	spec := imageSpec(pool, image)
	args := []string{"device", "map", spec}
	if confPath != "" {
		args = append(args, "--conf", confPath)
	}
	if keyringPath != "" && fileutils2.Exists(keyringPath) {
		args = append(args, "--keyring", keyringPath)
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("rbd", args...).Output()
	if err != nil {
		return "", errors.Wrapf(err, "rbd device map %s: %s", spec, string(out))
	}
	devStr := strings.TrimSpace(string(out))
	if devStr == "" {
		devPath, _, err := r.CheckConnect(diskPath)
		if err != nil || devPath == "" {
			return "", errors.Wrapf(err, "rbd map succeeded but device not found for %s", spec)
		}
		return r.checkPartition(devPath), nil
	}
	if !strings.HasPrefix(devStr, "/dev/") {
		devStr = "/dev/" + devStr
	}
	devStr = strings.TrimSpace(devStr)
	if idx := strings.Index(devStr, "\n"); idx > 0 {
		devStr = devStr[:idx]
	}
	return r.checkPartition(devStr), nil
}

func (r rbd) DisconnectDisk(diskPath string, mountPoint string) error {
	pool, image, confPath, keyringPath, err := parseRbdPath(diskPath)
	if err != nil {
		return err
	}
	spec := imageSpec(pool, image)
	mapped, err := r.listMappedDevices(confPath, keyringPath)
	if err != nil {
		log.Warningf("rbd device list before unmap: %v", err)
		return r.unmapBySpec(spec, confPath, keyringPath)
	}
	dev, ok := mapped[spec]
	if !ok {
		log.Infof("rbd image %s not mapped, skip unmap", spec)
		return nil
	}
	args := []string{"device", "unmap", dev}
	if confPath != "" {
		args = append(args, "--conf", confPath)
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("rbd", args...).Output()
	if err != nil {
		if strings.Contains(string(out), "not mapped") || strings.Contains(string(out), "No such device") {
			return nil
		}
		return r.unmapBySpec(spec, confPath, keyringPath)
	}
	log.Infof("rbd device unmap %s (image %s) ok", dev, spec)
	return nil
}

func (r rbd) unmapBySpec(spec, confPath, keyringPath string) error {
	args := []string{"device", "unmap", spec}
	if confPath != "" {
		args = append(args, "--conf", confPath)
	}
	if keyringPath != "" && fileutils2.Exists(keyringPath) {
		args = append(args, "--keyring", keyringPath)
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("rbd", args...).Output()
	if err != nil {
		return errors.Wrapf(err, "rbd device unmap %s: %s", spec, string(out))
	}
	return nil
}
