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
	"io"
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
	Encryption      bool
	Subformat       string
	IoLevel         TIONiceLevel
}

func NewQemuImage(path string) (*SQemuImage, error) {
	return NewEncryptedQemuImage(path, "")
}

func NewQemuImageWithIOLevel(path string, ioLevel TIONiceLevel) (*SQemuImage, error) {
	return NewEncryptedQemuImageWithIOLevel(path, "", IONiceNone)
}

func NewEncryptedQemuImage(path string, password string) (*SQemuImage, error) {
	return NewEncryptedQemuImageWithIOLevel(path, password, IONiceNone)
}

func NewEncryptedQemuImageWithIOLevel(path string, password string, ioLevel TIONiceLevel) (*SQemuImage, error) {
	qemuImg := SQemuImage{Path: path, Password: password, IoLevel: ioLevel}
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
		fileInfo, err := os.Stat(img.Path)
		if err != nil {
			if !os.IsNotExist(err) {
				return err
			} else {
				return nil
			}
		} else {
			img.ActualSizeBytes = fileInfo.Size()
		}
	}
	resp, err := func() (jsonutils.JSONObject, error) {
		output, err := procutils.NewRemoteCommandAsFarAsPossible(qemutils.GetQemuImg(), "info", img.Path, "--output", "json").Output()
		if err != nil {
			return nil, errors.Wrapf(err, "qemu-img info")
		}
		return jsonutils.Parse(output)
	}()
	if err != nil {
		return err
	}

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
				Compat        string `json:"compat"`
				LazyRefcounts int    `json:"lazy-refcounts"`
				RefcountBits  int    `json:"refcount-bits"`
				Corrupt       bool   `json:"corrupt"`
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
	img.Encryption = info.Encrypted
	img.BackFilePath = info.FullBackingFilename
	img.Subformat = info.CreateType

	if img.Format == RAW && fileutils2.IsFile(img.Path) {
		// test if it is an ISO
		blkType := fileutils2.GetBlkidType(img.Path)
		if utils.IsInStringArray(blkType, []string{"iso9660", "udf"}) {
			img.Format = ISO
		}
	}
	return nil
}

func (img *SQemuImage) IsValid() bool {
	return len(img.Format) > 0
}

func (img *SQemuImage) IsChained() bool {
	return len(img.BackFilePath) > 0
}

type SConvertInfo struct {
	Path     string
	Format   TImageFormat
	IoLevel  TIONiceLevel
	Password string
}

func Convert(srcInfo, destInfo SConvertInfo, options []string, compact bool, workerOpions []string) error {
	cmdline := []string{"-c", strconv.Itoa(int(srcInfo.IoLevel)),
		qemutils.GetQemuImg(), "convert"}
	if compact {
		cmdline = append(cmdline, "-c")
	}
	if workerOpions == nil {
		// https://bugzilla.redhat.com/show_bug.cgi?id=1969848
		// https://bugs.launchpad.net/qemu/+bug/1805256
		// qemu-img convert may hang on aarch64, fix: add -m 1
		cmdline = append(cmdline, "-m", "1")
	} else {
		cmdline = append(cmdline, workerOpions...)
	}
	cmdline = append(cmdline, "-f", srcInfo.Format.String(), "-O", destInfo.Format.String())
	if len(destInfo.Password) > 0 {
		if options == nil {
			options = make([]string, 0)
		}
		options = append(options, "encryption=on")
	}
	if len(options) > 0 {
		cmdline = append(cmdline, "-o", strings.Join(options, ","))
	}
	cmdline = append(cmdline, srcInfo.Path, destInfo.Path)
	log.Infof("XXXX qemu-img command: %s", cmdline)
	cmd := procutils.NewRemoteCommandAsFarAsPossible("ionice", cmdline...)
	var stdin io.WriteCloser
	var err error
	if len(srcInfo.Password) > 0 || len(destInfo.Password) > 0 {
		stdin, err = cmd.StdinPipe()
		if err != nil {
			return errors.Wrap(err, "convert stdin")
		}
	}
	err = cmd.Start()
	if err != nil {
		return errors.Wrap(err, "do convert")
	}
	if len(srcInfo.Password) > 0 || len(destInfo.Password) > 0 {
		input := ""
		if len(srcInfo.Password) > 0 {
			input = fmt.Sprintf("%s%s\r", input, srcInfo.Password)
		}
		if len(destInfo.Password) > 0 {
			input = fmt.Sprintf("%s%s\r", input, destInfo.Password)
		}
		io.WriteString(stdin, input+"\n")
	}
	err = cmd.Wait()
	if err != nil {
		log.Errorf("clone fail %s", err)
		os.Remove(destInfo.Path)
		return err
	}
	return nil
}

func (img *SQemuImage) doConvert(name string, format TImageFormat, options []string, compact bool, password string) error {
	if !img.IsValid() {
		return fmt.Errorf("self is not valid")
	}
	return Convert(SConvertInfo{
		Path:     img.Path,
		Format:   img.Format,
		IoLevel:  img.IoLevel,
		Password: img.Password,
	}, SConvertInfo{
		Path:     name,
		Format:   format,
		Password: password,
	}, options, compact, nil)
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

func (img *SQemuImage) clone(name string, format TImageFormat, options []string, compact bool, password string) (*SQemuImage, error) {
	err := img.doConvert(name, format, options, compact, password)
	if err != nil {
		return nil, err
	}
	return NewQemuImage(name)
}

func (img *SQemuImage) convert(format TImageFormat, options []string, compact bool, password string) error {
	tmpPath := fmt.Sprintf("%s.%s", img.Path, utils.GenRequestId(36))
	err := img.doConvert(tmpPath, format, options, compact, password)
	if err != nil {
		return err
	}
	cmd := procutils.NewCommand("mv", "-f", tmpPath, img.Path)
	err = cmd.Run()
	if err != nil {
		log.Errorf("convert move temp file error %s", err)
		os.Remove(tmpPath)
		return err
	}
	img.Password = password
	return img.parse()
}

func (img *SQemuImage) convertTo(
	format TImageFormat, options []string, compact bool, password string, output string,
) error {
	err := img.doConvert(output, format, options, compact, password)
	if err != nil {
		return err
	}
	img.Password = password
	return img.parse()
}

func (img *SQemuImage) Copy(name string) (*SQemuImage, error) {
	if !img.IsValid() {
		return nil, fmt.Errorf("self is not valid")
	}
	cmd := procutils.NewCommand("cp", "--sparse=always", img.Path, name)
	err := cmd.Run()
	if err != nil {
		log.Errorf("copy fail %s", err)
		os.Remove(name)
		return nil, err
	}
	return NewQemuImage(name)
}

func (img *SQemuImage) Convert2Qcow2To(output string, compact bool) error {
	options := make([]string, 0)
	// if len(backPath) > 0 {
	//	options = append(options, fmt.Sprintf("backing_file=%s", backPath))
	//} else
	if !compact {
		sparseOpts := qcow2SparseOptions()
		options = append(options, sparseOpts...)
	}
	return img.convertTo(QCOW2, options, compact, "", output)
}

func (img *SQemuImage) Convert2Qcow2(compact bool) error {
	options := make([]string, 0)
	// if len(backPath) > 0 {
	//	options = append(options, fmt.Sprintf("backing_file=%s", backPath))
	//} else
	if !compact {
		sparseOpts := qcow2SparseOptions()
		options = append(options, sparseOpts...)
	}
	return img.convert(QCOW2, options, compact, "")
}

func (img *SQemuImage) Convert2Vmdk(compact bool) error {
	return img.convert(VMDK, vmdkOptions(compact), compact, "")
}

func (img *SQemuImage) Convert2Vhd() error {
	return img.convert(VHD, nil, false, "")
}

func (img *SQemuImage) Convert2Raw() error {
	return img.convert(RAW, nil, false, "")
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
	return img.Convert2Qcow2(false)
}

func (img *SQemuImage) CloneQcow2(name string, compact bool) (*SQemuImage, error) {
	options := make([]string, 0)
	//if len(backPath) > 0 {
	//	options = append(options, fmt.Sprintf("backing_file=%s", backPath))
	//} else
	if !compact {
		sparseOpts := qcow2SparseOptions()
		options = append(options, sparseOpts...)
	}
	return img.clone(name, QCOW2, options, compact, "")
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
	return img.clone(name, VMDK, vmdkOptions(compact), compact, "")
}

func (img *SQemuImage) CloneVhd(name string) (*SQemuImage, error) {
	return img.clone(name, VHD, nil, false, "")
}

func (img *SQemuImage) CloneRaw(name string) (*SQemuImage, error) {
	return img.clone(name, RAW, nil, false, "")
}

func (img *SQemuImage) create(sizeMB int, format TImageFormat, options []string) error {
	if img.IsValid() {
		return fmt.Errorf("create: the image is valid??? %s", img.Format)
	}
	args := []string{"-c", strconv.Itoa(int(img.IoLevel)),
		qemutils.GetQemuImg(), "create", "-f", format.String()}
	if len(options) > 0 {
		args = append(args, "-o", strings.Join(options, ","))
	}
	args = append(args, img.Path)
	if sizeMB > 0 {
		args = append(args, fmt.Sprintf("%dM", sizeMB))
	}
	cmd := procutils.NewRemoteCommandAsFarAsPossible("ionice", args...)
	output, err := cmd.Output()
	if err != nil {
		log.Errorf("%v create error %s %s", args, output, err)
		return errors.Wrapf(err, "create image failed: %s", output)
	}
	return img.parse()
}

func (img *SQemuImage) CreateQcow2(sizeMB int, compact bool, backPath string) error {
	options := make([]string, 0)
	if len(backPath) > 0 {
		options = append(options, fmt.Sprintf("backing_file=%s", backPath))
		if !compact {
			options = append(options, "cluster_size=2M")
		}
	} else if !compact {
		sparseOpts := qcow2SparseOptions()
		if sizeMB <= 1024*1024*4 {
			options = append(options, "preallocation=metadata")
		}
		options = append(options, sparseOpts...)
	}
	return img.create(sizeMB, QCOW2, options)
}

func (img *SQemuImage) CreateVmdk(sizeMB int, compact bool) error {
	return img.create(sizeMB, VMDK, vmdkOptions(compact))
}

func (img *SQemuImage) CreateVhd(sizeMB int) error {
	return img.create(sizeMB, VHD, nil)
}

func (img *SQemuImage) CreateRaw(sizeMB int) error {
	return img.create(sizeMB, RAW, nil)
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
	cmd := procutils.NewRemoteCommandAsFarAsPossible("ionice", "-c", strconv.Itoa(int(img.IoLevel)),
		qemutils.GetQemuImg(), "resize", img.Path, fmt.Sprintf("%dM", sizeMB))
	err := cmd.Run()
	if err != nil {
		log.Errorf("resize fail %s", err)
		return err
	}
	return img.parse()
}

func (img *SQemuImage) Rebase(backPath string, force bool) error {
	if !img.IsValid() {
		return fmt.Errorf("self is not valid")
	}
	args := []string{"-c", strconv.Itoa(int(img.IoLevel)),
		qemutils.GetQemuImg(), "rebase"}
	if force {
		args = append(args, "-u")
	}
	args = append(args, "-b", backPath, img.Path)
	cmd := procutils.NewRemoteCommandAsFarAsPossible("ionice", args...)
	err := cmd.Run()
	if err != nil {
		log.Errorf("rebase fail %s", err)
		return err
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
	return cmd.Run()
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
