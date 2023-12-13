package lvmutils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

type LvNames struct {
	Report []struct {
		LV []struct {
			LVName string `json:"lv_name"`
		} `json:"lv"`
	} `json:"report"`
}

func GetLvNames(vg string) ([]string, error) {
	cmd := fmt.Sprintf("lvm lvs --reportformat json -o lv_name %s 2>/dev/null", vg)
	lvs, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrap(err, "lvm lvs")
	}
	var res LvNames
	err = json.Unmarshal(lvs, &res)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshal lvs")
	}
	if len(res.Report) != 1 {
		return nil, errors.Errorf("unexpect res %v", res)
	}
	lvNames := make([]string, len(res.Report[0].LV))
	for i := 0; i < len(res.Report[0].LV); i++ {
		lvNames = append(lvNames, res.Report[0].LV[i].LVName)
	}
	return lvNames, nil
}

type LvOrigin struct {
	Report []struct {
		LV []struct {
			Origin string `json:"origin"`
		} `json:"lv"`
	} `json:"report"`
}

func GetLvOrigin(lvPath string) (string, error) {
	cmd := fmt.Sprintf("lvm lvs --reportformat json -o origin %s 2>/dev/null", lvPath)
	lvs, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return "", errors.Wrap(err, "lvm lvs")
	}
	var res LvOrigin
	err = json.Unmarshal(lvs, &res)
	if err != nil {
		return "", errors.Wrap(err, "unmarshal lvs")
	}
	if len(res.Report) == 1 && len(res.Report[0].LV) == 1 {
		return res.Report[0].LV[0].Origin, nil

	}
	return "", errors.Errorf("unexpect res %v", res)
}

type VgProps struct {
	VgSize       int64
	VgFree       int64
	VgExtentSize int64
}

type VgReports struct {
	Report []struct {
		VG []struct {
			VgSize       string `json:"vg_size"`
			VgFree       string `json:"vg_free"`
			VgExtentSize string `json:"vg_extent_size"`
		} `json:"vg"`
	} `json:"report"`
}

func GetVgProps(vg string) (*VgProps, error) {
	cmd := fmt.Sprintf("lvm vgs --reportformat json -o vg_free,vg_size,vg_extent_size --units=B %s 2>/dev/null", vg)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "exec lvm command: %s", out)
	}
	var vgReports VgReports
	err = json.Unmarshal(out, &vgReports)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal vgprops %s", out)
	}
	if len(vgReports.Report) == 1 && len(vgReports.Report[0].VG) == 1 {
		var vgProps VgProps
		vgProps.VgExtentSize, err = strconv.ParseInt(strings.TrimSuffix(vgReports.Report[0].VG[0].VgExtentSize, "B"), 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parse extent size %s", vgReports.Report[0].VG[0].VgExtentSize)
		}
		vgProps.VgFree, err = strconv.ParseInt(strings.TrimSuffix(vgReports.Report[0].VG[0].VgFree, "B"), 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parse free size %s", vgReports.Report[0].VG[0].VgFree)
		}
		vgProps.VgSize, err = strconv.ParseInt(strings.TrimSuffix(vgReports.Report[0].VG[0].VgSize, "B"), 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "parse size %s", vgReports.Report[0].VG[0].VgSize)
		}

		return &vgProps, nil
	} else {
		return nil, errors.Errorf("invalid vg report %v", vgReports)
	}
}

func ExtendLvSize(vg string, size int64) (int64, error) {
	vgProps, err := GetVgProps(vg)
	if err != nil {
		return -1, err
	}
	extentSize := vgProps.VgExtentSize
	return (size + extentSize - 1) / extentSize * extentSize, nil
}

func LvCreate(vg, lv string, size int64) error {
	size, err := ExtendLvSize(vg, size)
	if err != nil {
		return err
	}

	out, err := procutils.NewRemoteCommandAsFarAsPossible(
		"lvm", "lvcreate", "--size", fmt.Sprintf("%dB", size), "-n", lv, vg, "-y",
	).Output()
	if err != nil {
		return errors.Wrapf(err, "LvCreate failed %s", out)
	}
	return nil
}

func LvCreateFromSnapshot(lv, snapShotPath string, size int64) error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible(
		"lvm", "lvcreate", "--size", fmt.Sprintf("%dB", size), "-n", lv, "-s", snapShotPath, "-y",
	).Output()
	if err != nil {
		return errors.Wrapf(err, "LvCreate from snapshot %s failed %s", snapShotPath, out)
	}
	return nil
}

// @param: lvPath string: should like /dev/<vg>/<lv>
func LvResize(vg, lvPath string, size int64) error {
	size, err := ExtendLvSize(vg, size)
	if err != nil {
		return err
	}

	out, err := procutils.NewRemoteCommandAsFarAsPossible(
		"lvm", "lvresize", "--size", fmt.Sprintf("%dB", size), lvPath, "-y",
	).Output()
	if err != nil {
		return errors.Wrapf(err, "LvResize failed %s", out)
	}
	return nil
}

// @param: lvPath string: should like /dev/<vg>/<lv>
func LvRemove(lvPath string) error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("lvm", "lvremove", lvPath, "-y").Output()
	if err != nil {
		return errors.Wrapf(err, "LvRemove failed %s", out)
	}
	return nil
}

// @param: dmPath string: should like /dev/mapper/<device>
func DmRemove(dmPath string) error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("dmsetup", "remove", dmPath).Output()
	if err != nil {
		return errors.Wrapf(err, "DmRemove failed %s", out)
	}
	return nil
}

func DmCreate(lv1, lv2, dmName string) error {
	var dmCreateScript = fmt.Sprintf(`
size1=$(blockdev --getsz $1)
size2=$(blockdev --getsz $2)
echo "0 $size1 linear $1 0
$size1 $size2 linear $2 0" | dmsetup create $3
`)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", dmCreateScript, "--", lv1, lv2, dmName).Output()
	if err != nil {
		return errors.Wrapf(err, "create device mapper failed %s", out)
	}
	return nil
}

func VgDisplay(vgName string) error {
	out, err := procutils.NewRemoteCommandAsFarAsPossible("lvm", "vgdisplay", vgName).Output()
	if err != nil {
		return errors.Wrapf(err, "vgdisplay %s failed %s", vgName, out)
	}
	return nil
}
