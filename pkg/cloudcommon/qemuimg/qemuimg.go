package qemuimg

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/onecloud/pkg/cloudcommon/qemutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
)

var (
	ErrUnsupportedFormat = errors.New("unsupported format")
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
}

func NewQemuImage(path string) (*SQemuImage, error) {
	return NewEncryptedQemuImage(path, "")
}

func NewEncryptedQemuImage(path string, password string) (*SQemuImage, error) {
	qemuImg := SQemuImage{Path: path, Password: password}
	err := qemuImg.parse()
	if err != nil {
		return nil, err
	}
	return &qemuImg, nil
}

func (img *SQemuImage) parse() error {
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
	} else if strings.HasPrefix(img.Path, models.STORAGE_RBD) {
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
	cmd := exec.Command(qemutils.GetQemuImg(), "info", img.Path)
	if len(img.Password) > 0 {
		cmd.Stdin = bytes.NewBuffer([]byte(img.Password))
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Errorf("qemu-img info %s fail %s", img.Path, err)
		return fmt.Errorf("qemu-img info error %s", err)
	}
	for {
		line, err := out.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(line, "file format:"):
				img.Format = TImageFormat(line[strings.LastIndexByte(line, ' ')+1:])
			case strings.HasPrefix(line, "virtual size:"):
				sizeStr := line[strings.LastIndexByte(line, '(')+1 : strings.LastIndexByte(line, ' ')]
				size, err := strconv.ParseInt(sizeStr, 10, -1)
				if err != nil {
					return fmt.Errorf("invalid size str %s", sizeStr)
				}
				img.SizeBytes = size
			case strings.HasPrefix(line, "cluster_size:"):
				sizeStr := line[strings.LastIndexByte(line, ' ')+1:]
				size, err := strconv.ParseInt(sizeStr, 10, -1)
				if err != nil {
					return fmt.Errorf("invalid cluster size str %s", sizeStr)
				}
				img.ClusterSize = int(size)
			case strings.HasPrefix(line, "backing file:"):
				img.BackFilePath = line[strings.LastIndexByte(line, ' ')+1:]
			case strings.HasPrefix(line, "compat:"):
				img.Compat = line[strings.LastIndexByte(line, ' ')+1:]
			case strings.HasPrefix(line, "encrypted:"):
				if line[strings.LastIndexByte(line, ' ')+1:] == "yes" {
					img.Encryption = true
				}
			case strings.HasPrefix(line, "create type:"):
				img.Subformat = line[strings.LastIndexByte(line, ' ')+1:]
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			} else {
				log.Errorf("read output fail %s", err)
				return fmt.Errorf("read output fail %s", err)
			}
		}
	}
	if img.Format == RAW {
		// test if it is an ISO
		blkType := fileutils2.GetBlkidType(img.Path)
		if blkType == "iso9660" {
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

func (img *SQemuImage) doConvert(name string, format TImageFormat, options []string, compact bool, password string) error {
	if !img.IsValid() {
		return fmt.Errorf("self is not valid")
	}
	cmdline := []string{"convert"}
	if compact {
		cmdline = append(cmdline, "-c")
	}
	cmdline = append(cmdline, "-f", img.Format.String(), "-O", format.String())
	if len(password) > 0 {
		if options == nil {
			options = make([]string, 0)
		}
		options = append(options, "encryption=on")
	}
	if len(options) > 0 {
		cmdline = append(cmdline, "-o", strings.Join(options, ","))
	}
	cmdline = append(cmdline, img.Path, name)
	cmd := exec.Command(qemutils.GetQemuImg(), cmdline...)
	if len(img.Password) > 0 || len(password) > 0 {
		input := ""
		if len(img.Password) > 0 {
			input = fmt.Sprintf("%s\r", input, img.Password)
		}
		if len(password) > 0 {
			input = fmt.Sprintf("%s%s\r", input, password)
		}
		cmd.Stdin = bytes.NewBuffer([]byte(input))
	}
	err := cmd.Run()
	if err != nil {
		log.Errorf("clone fail %s", err)
		os.Remove(name)
		return err
	}
	return nil
}

func (img *SQemuImage) Clone(name string, format TImageFormat, compact bool) (*SQemuImage, error) {
	switch format {
	case QCOW2:
		return img.CloneQcow2(name, compact)
	case VMDK:
		return img.CloneVmdk(name, compact)
	case RAW:
		return img.CloneRaw(name)
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
	cmd := exec.Command("mv", "-f", tmpPath, img.Path)
	err = cmd.Run()
	if err != nil {
		log.Errorf("convert move temp file error %s", err)
		os.Remove(tmpPath)
		return err
	}
	img.Password = password
	return img.parse()
}

func (img *SQemuImage) Copy(name string) (*SQemuImage, error) {
	if !img.IsValid() {
		return nil, fmt.Errorf("self is not valid")
	}
	cmd := exec.Command("cp", "--sparse=always", img.Path, name)
	err := cmd.Run()
	if err != nil {
		log.Errorf("copy fail %s", err)
		os.Remove(name)
		return nil, err
	}
	return NewQemuImage(name)
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
	return img.convert(VMDK, vmdkOptions(compact), false, "")
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

func (img *SQemuImage) CloneVmdk(name string, compact bool) (*SQemuImage, error) {
	return img.clone(name, VMDK, vmdkOptions(compact), compact, "")
}

func (img *SQemuImage) CloneRaw(name string) (*SQemuImage, error) {
	return img.clone(name, RAW, nil, false, "")
}

func (img *SQemuImage) create(sizeMB int, format TImageFormat, options []string) error {
	if img.IsValid() {
		return fmt.Errorf("The image is valid???")
	}
	args := []string{"create", "-f", format.String()}
	if len(options) > 0 {
		args = append(args, "-o", strings.Join(options, ","))
	}
	args = append(args, img.Path)
	if sizeMB > 0 {
		args = append(args, fmt.Sprintf("%dM", sizeMB))
	}
	cmd := exec.Command(qemutils.GetQemuImg(), args...)
	err := cmd.Run()
	if err != nil {
		log.Errorf("create error %s", err)
		return err
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
		options = append(options, sparseOpts...)
	}
	return img.create(sizeMB, QCOW2, options)
}

func (img *SQemuImage) CreateVmdk(sizeMB int, compact bool) error {
	return img.create(sizeMB, VMDK, vmdkOptions(compact))
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
	cmd := exec.Command(qemutils.GetQemuImg(), "resize", img.Path, fmt.Sprintf("%dM", sizeMB))
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
	args := []string{"rebase"}
	if force {
		args = append(args, "-u")
	}
	args = append(args, "-b", backPath, img.Path)
	cmd := exec.Command(qemutils.GetQemuImg(), args...)
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
	cmd := exec.Command("fallocate", "-l", fmt.Sprintf("%dm", img.GetSizeMB()), img.Path)
	return cmd.Run()
}

func (img *SQemuImage) String() string {
	return fmt.Sprintf("Qemu %s %d(%d) %s", img.Format, img.GetSizeMB(), img.GetActualSizeMB(), img.Path)
}
