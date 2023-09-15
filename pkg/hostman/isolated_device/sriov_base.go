package isolated_device

import (
	"fmt"
	"path"
	"strings"

	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type sSRIOVBaseDevice struct {
	*sBaseDevice
}

func ensureNumvfsEqualTotalvfs(devDir string) error {
	sriovNumvfs := path.Join(devDir, "sriov_numvfs")
	sriovTotalvfs := path.Join(devDir, "sriov_totalvfs")
	numvfs, err := fileutils2.FileGetContents(sriovNumvfs)
	if err != nil {
		return err
	}
	totalvfs, err := fileutils2.FileGetContents(sriovTotalvfs)
	if err != nil {
		return err
	}
	log.Infof("numvfs %s total vfs %s", numvfs, totalvfs)
	if numvfs != totalvfs {
		return fileutils2.FilePutContents(sriovNumvfs, fmt.Sprintf("%s", totalvfs), false)
	}
	return nil
}

func detectSRIOVDevice(vfBDF string) (*PCIDevice, error) {
	dev, err := detectPCIDevByAddrWithoutIOMMUGroup(vfBDF)
	if err != nil {
		return nil, err
	}
	driver, err := dev.getKernelDriver()
	if err != nil {
		return nil, err
	}
	if driver == VFIO_PCI_KERNEL_DRIVER {
		return dev, nil
	}
	if driver != "" {
		if err = dev.unbindDriver(); err != nil {
			return nil, err
		}
	}
	if err = dev.bindDriver(); err != nil {
		return nil, err
	}
	return dev, nil
}

func newSRIOVBaseDevice(dev *PCIDevice, devType string) *sSRIOVBaseDevice {
	return &sSRIOVBaseDevice{
		sBaseDevice: newBaseDevice(dev, devType),
	}
}

func (dev *sSRIOVBaseDevice) GetVGACmd() string {
	return ""
}

func (dev *sSRIOVBaseDevice) GetCPUCmd() string {
	return ""
}

func (dev *sSRIOVBaseDevice) GetWireId() string {
	return ""
}

func (dev *sSRIOVBaseDevice) GetQemuId() string {
	return fmt.Sprintf("dev_%s", strings.ReplaceAll(dev.GetAddr(), ":", "_"))
}

func (dev *sSRIOVBaseDevice) CustomProbe(idx int) error {
	// check environments on first probe
	if idx == 0 {
		for _, driver := range []string{"vfio", "vfio_iommu_type1", "vfio-pci"} {
			if err := procutils.NewRemoteCommandAsFarAsPossible("modprobe", driver).Run(); err != nil {
				return fmt.Errorf("modprobe %s: %v", driver, err)
			}
		}
	}

	driver, err := dev.GetKernelDriver()
	if err != nil {
		return fmt.Errorf("Nic %s is occupied by another driver: %s", dev.GetAddr(), driver)
	}
	return nil
}
