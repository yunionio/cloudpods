package validator

import (
	"fmt"

	"yunion.io/x/cloudmux/pkg/multicloud/azure/vhdcore/diskstream"
	"yunion.io/x/cloudmux/pkg/multicloud/azure/vhdcore/vhdfile"
)

// oneTB is one TeraByte
const oneTB int64 = 1024 * 1024 * 1024 * 1024

// ValidateVhd returns error if the vhdPath refer to invalid vhd.
func ValidateVhd(vhdPath string) error {
	vFactory := &vhdfile.FileFactory{}
	_, err := vFactory.Create(vhdPath)
	if err != nil {
		return fmt.Errorf("%s is not a valid VHD: %v", vhdPath, err)
	}
	return nil
}

// ValidateVhdSize returns error if size of the vhd referenced by vhdPath is more than
// the maximum allowed size (1TB)
func ValidateVhdSize(vhdPath string) error {
	stream, _ := diskstream.CreateNewDiskStream(vhdPath)
	if stream.GetSize() > oneTB {
		return fmt.Errorf("VHD size is too large ('%d'), maximum allowed size is '%d'", stream.GetSize(), oneTB)
	}
	return nil
}
