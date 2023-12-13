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

package qemuimg

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
	"yunion.io/x/onecloud/pkg/util/qemutils"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

var (
	ErrUnsupportedFormat = errors.Error("unsupported format")
)

type TIONiceLevel int

const (
	// The scheduling class. 0 for none, 1 for real time, 2 for best-effort, 3 for idle.
	IONiceNone       = TIONiceLevel(0)
	IONiceRealTime   = TIONiceLevel(1)
	IONiceBestEffort = TIONiceLevel(2)
	IONiceIdle       = TIONiceLevel(3)
)

type SQemuImage struct {
	Path            string
	Password        string
	Format          TImageFormat
	SizeBytes       int64
	ActualSizeBytes int64
	ClusterSize     int
	BackFilePath    string
	Compat          string
	Encrypted       bool
	Subformat       string
	IoLevel         TIONiceLevel

	EncryptFormat TEncryptFormat
	EncryptAlg    seclib2.TSymEncAlg
}

func NewQemuImage(path string) (*SQemuImage, error) {
	return NewQemuImageWithIOLevel(path, IONiceNone)
}

func NewQemuImageWithIOLevel(path string, ioLevel TIONiceLevel) (*SQemuImage, error) {
	qemuImg := SQemuImage{Path: path, IoLevel: ioLevel}
	err := qemuImg.parse()
	if err != nil {
		return nil, err
	}
	return &qemuImg, nil
}

func (img *SQemuImage) parse() error {
	if len(img.Path) == 0 {
		return fmt.Errorf("empty image path")
	}
	if strings.HasPrefix(img.Path, "nbd") {
		// nbd TCP -> nbd:<server-ip>:<port>
		// nbd Unix Domain Sockets -> nbd:unix:<domain-socket-file>
		img.ActualSizeBytes = 0
	} else if strings.HasPrefix(img.Path, "iscsi") {
		// iSCSI LUN -> iscsi://<target-ip>[:<port>]/<target-iqn>/<lun>
		return cloudprovider.ErrNotImplemented
	} else if strings.HasPrefix(img.Path, "sheepdog") {
		// sheepdog -> sheepdog[+tcp|+unix]://[host:port]/vdiname[?socket=path][#snapid|#tag]
		return cloudprovider.ErrNotImplemented
	} else if strings.HasPrefix(img.Path, api.STORAGE_RBD) {
		img.ActualSizeBytes = 0
	} else {
		// check file existence
		fileInfo, err := procutils.RemoteStat(img.Path)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			} else {
				// not created yet
				return nil
			}
		} else {
			img.ActualSizeBytes = fileInfo.Size()
		}
	}
	resp, err := func() (jsonutils.JSONObject, error) {
		output, err := procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuImg(), "info", "-U", img.Path, "--output", "json").Output()
		if err != nil {
			return nil, errors.Wrapf(err, "qemu-img info: %s", output)
		}
		return jsonutils.Parse(output)
	}()
	if err != nil {
		return err
	}
	/*
	   {
	       "virtual-size": 107374182400,
	       "filename": "/opt/cloud/workspace/data/glance/images/3ccecd2b-0ab4-4525-8e64-6f1d3c3a2457",
	       "cluster-size": 65536,
	       "format": "vmdk",
	       "actual-size": 2173186048,
	       "format-specific": {
	           "type": "vmdk",
	           "data": {
	               "cid": 3046516340,
	               "parent-cid": 4294967295,
	               "create-type": "streamOptimized",
	               "extents": [
	                   {
	                       "compressed": true,
	                       "virtual-size": 107374182400,
	                       "filename": "/opt/cloud/workspace/data/glance/images/3ccecd2b-0ab4-4525-8e64-6f1d3c3a2457",
	                       "cluster-size": 65536,
	                       "format": ""
	                   }
	               ]
	           }
	       },
	       "dirty-flag": false
	   }
	*/

	info := struct {
		VirtualSizeBytes      int64  `json:"virtual-size"`
		Filename              string `json:"filename"`
		Format                string `json:"format"`
		ActualSizeBytes       int64  `json:"actual-size"`
		ClusterSize           int    `json:"cluster-size"`
		BackingFilename       string `json:"backing-filename"`
		FullBackingFilename   string `json:"full-backing-filename"`
		BackingFilenameFormat string `json:"backing-filename-format"`
		Encrypted             bool   `json:"encrypted"`
		FormatSpecific        struct {
			Type string `json:"type"`
			Data struct {
				Cid        uint64 `json:"cid"`
				ParentCid  uint64 `json:"parent-cid"`
				CreateType string `json:"create-type"`
				Extents    []struct {
					Compressed  bool   `json:"compressed"`
					VirtualSize uint64 `json:"virtual-size"`
					Filename    string `json:"filename"`
					ClusterSize uint64 `json:"cluster-size"`
					Format      string `json:"format"`
				} `json:"extents"`
				Compat        string `json:"compat"`
				LazyRefcounts int    `json:"lazy-refcounts"`
				RefcountBits  int    `json:"refcount-bits"`
				Corrupt       bool   `json:"corrupt"`
				Encrypt       struct {
					IvgenAlg  string `json:"ivgen-alg"`
					HashAlg   string `json:"hash-alg"`
					CipherAlg string `json:"cipher-alg"`
					Uuid      string `json:"uuid"`
					Format    string `json:"format"`
					CipherMod string `json:"cipher-mode"`
				} `json:"encrypt"`
			} `json:"data"`
		} `json:"format-specific"`
		CreateType string `json:"create-type"`
		DirtyFlag  bool   `json:"dirty-flag"`
	}{}

	err = resp.Unmarshal(&info)
	if err != nil {
		return errors.Wrapf(err, "resp.Unmarshal")
	}
	img.Format = TImageFormat(info.Format)
	img.SizeBytes = info.VirtualSizeBytes
	img.ClusterSize = info.ClusterSize
	img.Compat = info.FormatSpecific.Data.Compat
	img.Encrypted = info.Encrypted
	img.BackFilePath, err = ParseQemuFilepath(info.FullBackingFilename)
	if err != nil {
		return errors.Wrap(err, "ParseQemuFilepath")
	}
	img.Subformat = info.CreateType
	if img.Subformat == "" {
		img.Subformat = info.FormatSpecific.Data.CreateType
	}

	if img.Encrypted {
		img.EncryptFormat = TEncryptFormat(info.FormatSpecific.Data.Encrypt.Format)
		img.EncryptAlg = seclib2.TSymEncAlg(info.FormatSpecific.Data.Encrypt.CipherAlg)
	}

	if img.Format == RAW && fileutils2.IsFile(img.Path) {
		// test if it is an ISO
		blkType := fileutils2.GetBlkidType(img.Path)
		if utils.IsInStringArray(blkType, []string{"iso9660", "udf"}) {
			img.Format = ISO
		}
	}
	return nil
}

// base64 password
func (img *SQemuImage) SetPassword(password string) {
	img.Password = password
}

func (img *SQemuImage) IsValid() bool {
	return len(img.Format) > 0
}

func (img *SQemuImage) IsChained() bool {
	return len(img.BackFilePath) > 0
}

func (img *SQemuImage) GetBackingChain() ([]string, error) {
	if len(img.BackFilePath) > 0 {
		backImg, err := NewQemuImage(img.BackFilePath)
		if err != nil {
			return nil, err
		}
		backingChain, err := backImg.GetBackingChain()
		if err != nil {
			return nil, err
		}
		return append(backingChain, img.BackFilePath), nil
	} else {
		return []string{}, nil
	}
}

type TEncryptFormat string

const (
	EncryptFormatLuks = "luks"
)

type SImageInfo struct {
	Path     string
	Format   TImageFormat
	IoLevel  TIONiceLevel
	Password string

	// only luks supported
	EncryptFormat TEncryptFormat
	// aes-256, sm4
	EncryptAlg seclib2.TSymEncAlg
	secId      string
}

func (info *SImageInfo) SetSecId(id string) {
	info.secId = id
}

func (info SImageInfo) ImageOptions() string {
	opts := make([]string, 0)
	format := info.Format
	if len(format) == 0 {
		format = QCOW2
	}
	opts = append(opts, fmt.Sprintf("driver=%s", format))
	opts = append(opts, fmt.Sprintf("file.filename=%s", info.Path))
	if info.Encrypted() {
		encFormat := info.EncryptFormat
		if len(encFormat) == 0 {
			encFormat = EncryptFormatLuks
		}
		opts = append(opts, fmt.Sprintf("encrypt.format=%s", encFormat))
		secId := info.secId
		if len(secId) == 0 {
			secId = "sec0"
		}
		opts = append(opts, fmt.Sprintf("encrypt.key-secret=%s", secId))
		// if info.EncryptFormat == EncryptFormatLuks {
		//	opts = append(opts, fmt.Sprintf("encrypt.cipher-alg=%s", info.EncryptAlg))
		// }
	}
	return strings.Join(opts, ",")
}

func (info SImageInfo) SecretOptions() string {
	if info.Encrypted() {
		opts := make([]string, 0)
		secId := info.secId
		if len(secId) == 0 {
			secId = "sec0"
		}
		opts = append(opts, "secret")
		opts = append(opts, fmt.Sprintf("id=%s", secId))
		opts = append(opts, fmt.Sprintf("data=%s", info.Password))
		opts = append(opts, "format=base64")

		return strings.Join(opts, ",")
	}
	return ""
}

func (info SImageInfo) Encrypted() bool {
	return len(info.Password) > 0
}

func Convert(srcInfo, destInfo SImageInfo, compact bool, workerOpions []string) error {
	if srcInfo.Encrypted() || destInfo.Encrypted() {
		return convertEncrypt(srcInfo, destInfo, compact, workerOpions)
	} else {
		return convertOther(srcInfo, destInfo, compact, workerOpions)
	}
}

func convertOther(srcInfo, destInfo SImageInfo, compact bool, workerOpions []string) error {
	cmdline := []string{"-c", strconv.Itoa(int(srcInfo.IoLevel)),
		qemutils.GetQemuImg(), "convert"}
	if compact {
		cmdline = append(cmdline, "-c")
	}
	if workerOpions == nil {
		// https://bugzilla.redhat.com/show_bug.cgi?id=1969848
		// https://bugs.launchpad.net/qemu/+bug/1805256
		// qemu-img convert may hang on aarch64, fix: add -m 1
		// no need to limit 1 any more, the bug has been fixed! - QIUJIAN
		// cmdline = append(cmdline, "-m", "1")
	} else {
		cmdline = append(cmdline, workerOpions...)
	}
	if compact {
		cmdline = append(cmdline, "-c")
	}
	cmdline = append(cmdline, "-f", srcInfo.Format.String(), "-O", destInfo.Format.String())
	if destInfo.Format.String() == "vmdk" { // for esxi vmdk
		cmdline = append(cmdline, "-o")
		cmdline = append(cmdline, vmdkOptions(compact)...)
	}
	cmdline = append(cmdline, srcInfo.Path, destInfo.Path)
	log.Infof("XXXX qemu-img command: %s", cmdline)
	cmd := procutils.NewRemoteCommandAsFarAsPossible("ionice", cmdline...)
	output, err := cmd.Output()
	if err != nil {
		os.Remove(destInfo.Path)
		return errors.Wrapf(err, "convert: %s", output)
	}
	return nil
}

func convertEncrypt(srcInfo, destInfo SImageInfo, compact bool, workerOpions []string) error {
	source, err := NewQemuImageWithIOLevel(srcInfo.Path, srcInfo.IoLevel)
	if err != nil {
		return errors.Wrapf(err, "NewQemuImage source %s", srcInfo.Path)
	}
	target, err := NewQemuImage(destInfo.Path)
	if err != nil {
		return errors.Wrapf(err, "NewQemuImage dest %s", destInfo.Path)
	}
	err = target.CreateQcow2(source.GetSizeMB(), compact, "", destInfo.Password, destInfo.EncryptFormat, destInfo.EncryptAlg)
	if err != nil {
		return errors.Wrapf(err, "Create target image %s", destInfo.Path)
	}
	cmdline := []string{"-c", strconv.Itoa(int(srcInfo.IoLevel)), qemutils.GetQemuImg(), "convert"}
	if compact {
		cmdline = append(cmdline, "-c")
	}
	if workerOpions == nil {
		// https://bugzilla.redhat.com/show_bug.cgi?id=1969848
		// https://bugs.launchpad.net/qemu/+bug/1805256
		// qemu-img convert may hang on aarch64, fix: add -m 1
		// no need to limit 1 any more, the bug has been fixed! - QIUJIAN
		// cmdline = append(cmdline, "-m", "1")
	} else {
		cmdline = append(cmdline, workerOpions...)
	}
	if srcInfo.Encrypted() {
		if srcInfo.Format != QCOW2 {
			return errors.Wrap(errors.ErrNotSupported, "source image not support encryption")
		}
	}
	if destInfo.Encrypted() {
		if destInfo.Format != QCOW2 {
			return errors.Wrap(errors.ErrNotSupported, "target image not support encryption")
		}
	}
	if srcInfo.Encrypted() && destInfo.Encrypted() {
		if srcInfo.Password == destInfo.Password {
			srcInfo.secId = "sec0"
			destInfo.secId = "sec0"
			cmdline = append(cmdline, "--object", srcInfo.SecretOptions())
		} else {
			srcInfo.secId = "sec0"
			cmdline = append(cmdline, "--object", srcInfo.SecretOptions())
			destInfo.secId = "sec1"
			cmdline = append(cmdline, "--object", destInfo.SecretOptions())
		}
	} else if srcInfo.Encrypted() {
		srcInfo.secId = "sec0"
		cmdline = append(cmdline, "--object", srcInfo.SecretOptions())
	} else if destInfo.Encrypted() {
		destInfo.secId = "sec0"
		cmdline = append(cmdline, "--object", destInfo.SecretOptions())
	} else {
		// dead branch
	}
	cmdline = append(cmdline, "--image-opts", srcInfo.ImageOptions())
	cmdline = append(cmdline, "--target-image-opts", destInfo.ImageOptions())
	cmdline = append(cmdline, "-n")
	cmd := procutils.NewRemoteCommandAsFarAsPossible("ionice", cmdline...)
	output, err := cmd.Output()
	log.Infof("XXXX qemu-img convert command: %s output: %s", cmdline, output)
	if err != nil {
		os.Remove(destInfo.Path)
		return errors.Wrapf(err, "convert: %s", string(output))
	}
	return nil
}

func (img *SQemuImage) doConvert(targetPath string, format TImageFormat, compact bool, password string, encryptFormat TEncryptFormat, encryptAlg seclib2.TSymEncAlg) error {
	if !img.IsValid() {
		return fmt.Errorf("self is not valid")
	}
	return Convert(SImageInfo{
		Path:     img.Path,
		Format:   img.Format,
		IoLevel:  img.IoLevel,
		Password: img.Password,

		EncryptFormat: img.EncryptFormat,
		EncryptAlg:    img.EncryptAlg,
	}, SImageInfo{
		Path:     targetPath,
		Format:   format,
		Password: password,

		EncryptFormat: encryptFormat,
		EncryptAlg:    encryptAlg,
	}, compact, nil)
}

func (img *SQemuImage) Clone(name string, format TImageFormat, compact bool) (*SQemuImage, error) {
	switch format {
	case QCOW2:
		return img.CloneQcow2(name, compact)
	case VMDK:
		return img.CloneVmdk(name, compact)
	case RAW:
		return img.CloneRaw(name)
	case VHD:
		return img.CloneVhd(name)
	default:
		return nil, ErrUnsupportedFormat
	}
}

func (img *SQemuImage) clone(target string, format TImageFormat, compact bool, password string, encryptFormat TEncryptFormat, encryptAlg seclib2.TSymEncAlg) (*SQemuImage, error) {
	err := img.doConvert(target, format, compact, password, encryptFormat, encryptAlg)
	if err != nil {
		return nil, errors.Wrap(err, "doConvert")
	}
	return NewQemuImage(target)
}

func (img *SQemuImage) convert(format TImageFormat, compact bool, password string, encryptFormat TEncryptFormat, alg seclib2.TSymEncAlg) error {
	tmpPath := fmt.Sprintf("%s.%s", img.Path, utils.GenRequestId(36))
	err := img.doConvert(tmpPath, format, compact, password, encryptFormat, alg)
	if err != nil {
		return errors.Wrap(err, "doConvert")
	}
	cmd := procutils.NewRemoteCommandAsFarAsPossible("mv", "-f", tmpPath, img.Path)
	output, err := cmd.Output()
	if err != nil {
		os.Remove(tmpPath)
		return errors.Wrapf(err, "move %s", string(output))
	}
	img.Password = password
	img.EncryptFormat = encryptFormat
	img.EncryptAlg = alg
	return img.parse()
}

func (img *SQemuImage) convertTo(
	format TImageFormat, compact bool, password string, output string, encFormat TEncryptFormat, encAlg seclib2.TSymEncAlg,
) error {
	err := img.doConvert(output, format, compact, password, encFormat, encAlg)
	if err != nil {
		return errors.Wrap(err, "doConvert")
	}
	img.Password = password
	return img.parse()
}

func (img *SQemuImage) Copy(name string) (*SQemuImage, error) {
	if !img.IsValid() {
		return nil, fmt.Errorf("self is not valid")
	}
	cmd := procutils.NewRemoteCommandAsFarAsPossible("cp", "--sparse=always", img.Path, name)
	output, err := cmd.Output()
	if err != nil {
		os.Remove(name)
		return nil, errors.Wrapf(err, "cp: %s", string(output))
	}
	newImg, err := NewQemuImage(name)
	if err != nil {
		return nil, errors.Wrap(err, "NewQemuImage")
	}
	newImg.Password = img.Password
	return newImg, nil
}

func (img *SQemuImage) Convert2Qcow2To(output string, compact bool, password string, encFormat TEncryptFormat, encAlg seclib2.TSymEncAlg) error {
	return img.convertTo(QCOW2, compact, password, output, encFormat, encAlg)
}

func (img *SQemuImage) Convert2Qcow2(compact bool, password string, encFormat TEncryptFormat, encAlg seclib2.TSymEncAlg) error {
	if len(password) == 0 && len(img.Password) > 0 {
		password = img.Password
		encFormat = img.EncryptFormat
		encAlg = img.EncryptAlg
	}
	return img.convert(QCOW2, compact, password, encFormat, encAlg)
}

func (img *SQemuImage) Convert2Vmdk(compact bool) error {
	return img.convert(VMDK, compact, "", "", "")
}

func (img *SQemuImage) Convert2Vhd() error {
	return img.convert(VHD, false, "", "", "")
}

func (img *SQemuImage) Convert2Raw() error {
	return img.convert(RAW, false, "", "", "")
}

func (img *SQemuImage) IsRaw() bool {
	return img.Format == RAW
}

func (img *SQemuImage) IsSparseQcow2() bool {
	return img.Format == QCOW2 && img.ClusterSize >= 1024*1024*2
}

func (img *SQemuImage) IsSparseVmdk() bool {
	return img.Format == VMDK && img.Subformat != "streamOptimized"
}

func (img *SQemuImage) IsSparse() bool {
	return img.IsRaw() || img.IsSparseQcow2() || img.IsSparseVmdk()
}

func (img *SQemuImage) Expand() error {
	if img.IsSparse() {
		return nil
	}
	return img.Convert2Qcow2(false, img.Password, img.EncryptFormat, img.EncryptAlg)
}

func (img *SQemuImage) CloneQcow2(name string, compact bool) (*SQemuImage, error) {
	return img.clone(name, QCOW2, compact, img.Password, img.EncryptFormat, img.EncryptAlg)
}

func vmdkOptions(compact bool) []string {
	if compact {
		return []string{"subformat=streamOptimized"}
	} else {
		return []string{"subformat=monolithicSparse"}
	}
}

// func vhdOptions(compact bool) []string {
//	if compact {
//		return []string{"subformat=dynamic"}
//	} else {
//		return []string{"subformat=fixed"}
//	}
// }

func (img *SQemuImage) CloneVmdk(name string, compact bool) (*SQemuImage, error) {
	return img.clone(name, VMDK, compact, "", "", "")
}

func (img *SQemuImage) CloneVhd(name string) (*SQemuImage, error) {
	return img.clone(name, VHD, false, "", "", "")
}

func (img *SQemuImage) CloneRaw(name string) (*SQemuImage, error) {
	return img.clone(name, RAW, false, "", "", "")
}

func (img *SQemuImage) create(sizeMB int, format TImageFormat, options []string, extraArgs []string) error {
	if img.IsValid() && img.Format != RAW {
		return fmt.Errorf("create: the image is valid??? %s", img.Format)
	}
	args := []string{"-c", strconv.Itoa(int(img.IoLevel)),
		qemutils.GetQemuImg(), "create", "-f", format.String()}
	if len(options) > 0 {
		args = append(args, "-o", strings.Join(options, ","))
	}
	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}
	args = append(args, img.Path)
	if sizeMB > 0 {
		args = append(args, fmt.Sprintf("%dM", sizeMB))
	}
	cmd := procutils.NewRemoteCommandAsFarAsPossible("ionice", args...)
	output, err := cmd.Output()
	if err != nil {
		return errors.Wrapf(err, "create image failed: %s", output)
	}
	return img.parse()
}

type SFileInfo struct {
	Driver   string `json:"driver"`
	Filename string `json:"filename"`
	Locking  string `json:"locking"`
}

type SQcow2FileInfo struct {
	Driver           string    `json:"driver"`
	EncryptKeySecret string    `json:"encrypt.key-secret"`
	EncryptFormat    string    `json:"encrypt.format"`
	File             SFileInfo `json:"file"`
}

func newQcow2FileInfo(filePath string) SQcow2FileInfo {
	return SQcow2FileInfo{
		Driver: "qcow2",
		File: SFileInfo{
			Driver:   "file",
			Filename: filePath,
			Locking:  "off",
		},
	}
}

func GetQemuFilepath(path string, encKey string, encFormat TEncryptFormat) string {
	if len(encKey) == 0 {
		return path
	}
	info := newQcow2FileInfo(path)
	info.EncryptKeySecret = encKey
	info.EncryptFormat = string(encFormat)
	return fmt.Sprintf("json:%s", jsonutils.Marshal(info))
}

func (img *SQemuImage) CreateQcow2(sizeMB int, compact bool, backPath string, password string, encFormat TEncryptFormat, encAlg seclib2.TSymEncAlg) error {
	options := make([]string, 0)
	extraArgs := make([]string, 0)
	if len(password) > 0 {
		extraArgs = append(extraArgs, "--object", fmt.Sprintf("secret,id=sec0,data=%s,format=base64", password))
	}
	if len(backPath) > 0 {
		// options = append(options, fmt.Sprintf("backing_file=%s", backPath))
		backQemu, err := NewQemuImage(backPath)
		if err != nil {
			return errors.Wrap(err, "parse backing file")
		}
		if backQemu.Encrypted {
			extraArgs = append(extraArgs, "-b", GetQemuFilepath(backPath, "sec0", EncryptFormatLuks))
		} else {
			extraArgs = append(extraArgs, "-b", backPath)
		}
		if !compact {
			options = append(options, "cluster_size=2M")
		}
		if sizeMB == 0 {
			sizeMB = backQemu.GetSizeMB()
		}
	} else if !compact {
		sparseOpts := qcow2SparseOptions()
		if sizeMB <= 1024*1024*4 {
			options = append(options, "preallocation=metadata")
		}
		options = append(options, sparseOpts...)
	}
	if len(password) > 0 {
		if len(encFormat) == 0 {
			encFormat = EncryptFormatLuks
		}
		options = append(options, fmt.Sprintf("encrypt.format=%s", encFormat))
		options = append(options, fmt.Sprintf("encrypt.key-secret=sec0"))
		if encFormat == EncryptFormatLuks {
			options = append(options, fmt.Sprintf("encrypt.cipher-alg=%s", encAlg))
		}
	}
	return img.create(sizeMB, QCOW2, options, extraArgs)
}

func (img *SQemuImage) CreateVmdk(sizeMB int, compact bool) error {
	return img.create(sizeMB, VMDK, vmdkOptions(compact), nil)
}

func (img *SQemuImage) CreateVhd(sizeMB int) error {
	return img.create(sizeMB, VHD, nil, nil)
}

func (img *SQemuImage) CreateRaw(sizeMB int) error {
	return img.create(sizeMB, RAW, nil, nil)
}

func (img *SQemuImage) GetSizeMB() int {
	return int(img.SizeBytes / 1024 / 1024)
}

func (img *SQemuImage) GetActualSizeMB() int {
	return int(img.ActualSizeBytes / 1024 / 1024)
}

func (img *SQemuImage) Resize(sizeMB int) error {
	if !img.IsValid() {
		return fmt.Errorf("self is not valid")
	}
	encInfo := SImageInfo{
		Path:    img.Path,
		Format:  img.Format,
		IoLevel: img.IoLevel,
	}
	args := make([]string, 0)
	args = append(args, "-c", strconv.Itoa(int(img.IoLevel)), qemutils.GetQemuImg(), "resize")
	if len(img.Password) > 0 {
		encInfo.Password = img.Password
		encInfo.EncryptFormat = img.EncryptFormat
		encInfo.EncryptAlg = img.EncryptAlg
		encInfo.secId = "sec0"
		args = append(args, "--object", encInfo.SecretOptions())
	}
	args = append(args, "--image-opts", encInfo.ImageOptions())
	args = append(args, fmt.Sprintf("%dM", sizeMB))
	cmd := procutils.NewRemoteCommandAsFarAsPossible("ionice", args...)
	output, err := cmd.Output()
	if err != nil {
		return errors.Wrapf(err, "resize: %s", string(output))
	}
	return img.parse()
}

func (img *SQemuImage) Rebase(backPath string, force bool) error {
	if !img.IsValid() {
		return fmt.Errorf("self is not valid")
	}
	encInfo := SImageInfo{
		Path:    img.Path,
		Format:  img.Format,
		IoLevel: img.IoLevel,
	}
	args := []string{"-c", strconv.Itoa(int(img.IoLevel)),
		qemutils.GetQemuImg(), "rebase"}
	if len(img.Password) > 0 {
		encInfo.Password = img.Password
		encInfo.EncryptFormat = img.EncryptFormat
		encInfo.EncryptAlg = img.EncryptAlg
		encInfo.secId = "sec0"
		args = append(args, "--object", encInfo.SecretOptions())
	}
	args = append(args, "--image-opts", encInfo.ImageOptions())
	if force {
		args = append(args, "-u")
	}
	args = append(args, "-b", backPath)
	cmd := procutils.NewRemoteCommandAsFarAsPossible("ionice", args...)
	output, err := cmd.Output()
	if err != nil {
		return errors.Wrapf(err, "rebase %s", string(output))
	}
	return img.parse()
}

func (img *SQemuImage) Delete() error {
	if !img.IsValid() {
		return nil
	}
	err := os.Remove(img.Path)
	if err != nil {
		log.Errorf("delete fail %s", err)
		return err
	}
	img.Format = ""
	img.ActualSizeBytes = 0
	img.SizeBytes = 0
	return nil
}

func (img *SQemuImage) Fallocate() error {
	if !img.IsValid() {
		return fmt.Errorf("self is not valid")
	}
	cmd := procutils.NewCommand("fallocate", "-l", fmt.Sprintf("%dm", img.GetSizeMB()), img.Path)
	output, err := cmd.Output()
	if err != nil {
		return errors.Wrapf(err, "fallocate: %s", string(output))
	}
	return nil
}

func (img *SQemuImage) String() string {
	return fmt.Sprintf("Qemu %s %d(%d) %s", img.Format, img.GetSizeMB(), img.GetActualSizeMB(), img.Path)
}

func (img *SQemuImage) WholeChainFormatIs(format string) (bool, error) {
	if img.Format.String() != format {
		return false, nil
	}
	if len(img.BackFilePath) > 0 {
		backImg, err := NewQemuImage(img.BackFilePath)
		if err != nil {
			return false, err
		}
		return backImg.WholeChainFormatIs(format)
	}
	return true, nil
}

func (img *SQemuImage) Check() error {
	args := []string{"-c", strconv.Itoa(int(img.IoLevel)), qemutils.GetQemuImg(), "check"}
	info := SImageInfo{
		Path:          img.Path,
		Format:        img.Format,
		IoLevel:       img.IoLevel,
		Password:      img.Password,
		EncryptFormat: img.EncryptFormat,
		EncryptAlg:    img.EncryptAlg,
		secId:         "sec0",
	}
	if info.Encrypted() {
		args = append(args, "--object", info.SecretOptions())
	}
	args = append(args, "--image-opts", info.ImageOptions())
	cmd := procutils.NewRemoteCommandAsFarAsPossible("ionice", args...)
	output, err := cmd.Output()
	if err != nil {
		return errors.Wrapf(err, "check: %s", string(output))
	}
	return nil
}

// "json:{\"driver\":\"qcow2\",\"file\":{\"driver\":\"file\",\"filename\":\"/opt/cloud/workspace/disks/snapshots/72a2383d-e980-486f-816c-6c562e1757f3_snap/f39f225a-921f-492e-8fb6-0a4167d6ed91\"}}"
func ParseQemuFilepath(pathInfo string) (string, error) {
	if strings.HasPrefix(pathInfo, "json:{") {
		pathJson, err := jsonutils.ParseString(pathInfo[len("json:"):])
		if err != nil {
			return "", errors.Wrap(err, "jsonutils.ParseString")
		}
		path, err := pathJson.GetString("file", "filename")
		if err != nil {
			return "", errors.Wrap(err, "GetString file.filename")
		}
		return path, nil
	} else {
		return pathInfo, nil
	}
}
