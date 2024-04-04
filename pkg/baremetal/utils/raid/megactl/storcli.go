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

package megactl

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
)

type StorcliAdaptor struct {
	Controller   int
	isSNEmpty    bool
	sn           string
	name         string
	busNumber    string
	deviceNumber string
	funcNumber   string
}

func newStorcliAdaptor() *StorcliAdaptor {
	return &StorcliAdaptor{
		Controller:   -1,
		isSNEmpty:    false,
		sn:           "",
		name:         "",
		busNumber:    "",
		deviceNumber: "",
		funcNumber:   "",
	}
}

func (a StorcliAdaptor) key() string {
	return a.name + a.sn
}

func (a *StorcliAdaptor) isComplete() bool {
	if a.isSNEmpty {
		return a.Controller >= 0 && a.name != "" && a.isPciInfoFilled()
	}
	return a.Controller >= 0 && a.name != "" && a.sn != "" && a.isPciInfoFilled()
}

func (a *StorcliAdaptor) isPciInfoFilled() bool {
	return a.busNumber != "" && a.deviceNumber != "" && a.funcNumber != ""
}

func parseLineForStorcli(a *StorcliAdaptor, l string) {
	controllerKey := "Controller"
	productNameKey := "Product Name"
	snKey := "Serial Number"
	busNumber := "Bus Number"
	devNumber := "Device Number"
	funcNumber := "Function Number"

	if !regexp.MustCompile(fmt.Sprintf("^(%s|%s|%s|%s|%s|%s)\\s*=", controllerKey, productNameKey, snKey, busNumber, devNumber, funcNumber)).Match([]byte(l)) {
		return
	}

	parts := strings.Split(l, "=")
	if len(parts) != 2 {
		return
	}

	trimBeginEndSpace := func(s string) string {
		return strings.TrimLeft(strings.TrimRight(s, " "), " ")
	}

	key := trimBeginEndSpace(parts[0])
	val := trimBeginEndSpace(parts[1])

	if key == snKey && val == "" {
		a.isSNEmpty = true
		return
	}

	switch key {
	case controllerKey:
		a.Controller, _ = strconv.Atoi(val)
	case snKey:
		a.sn = val
	case productNameKey:
		a.name = val
	case busNumber:
		a.busNumber = fmt.Sprintf("0%s", val)
	case devNumber:
		a.deviceNumber = fmt.Sprintf("0%s", val)
	case funcNumber:
		a.funcNumber = val
	}
}

func (a *StorcliAdaptor) parseLine(l string) {
	parseLineForStorcli(a, l)
}

func (a *StorcliAdaptor) String() string {
	return fmt.Sprintf("{controller: %d, isSNEmpty: %v, sn: %q, name: %s}", a.Controller, a.isSNEmpty, a.sn, a.name)
}

func (a *StorcliAdaptor) getPhyDevs(getCmd func(...string) string, term raid.IExecTerm) ([]*StorcliPhysicalDrive, error) {
	cmd := getCmd(fmt.Sprintf("/c%d/eall/sall show J", a.Controller))
	lines, err := term.Run(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "Get storcli PDList json output")
	}
	jStr := strings.Join(lines, "\n")
	info, err := parseStorcliControllers(jStr)
	if err != nil {
		return nil, errors.Wrapf(err, "parseStorcliControllers by string %q", jStr)
	}
	if len(info.Controllers) != 1 {
		return nil, errors.Errorf("Not matched storcli PD list: %s", jsonutils.Marshal(info).PrettyString())
	}
	return info.Controllers[0].ResponseData.Info, nil
}

func (a *StorcliAdaptor) getMegaPhyDevs(getCmd func(...string) string, term raid.IExecTerm) ([]*MegaRaidPhyDev, error) {
	pdList, err := a.getPhyDevs(getCmd, term)
	if err != nil {
		return nil, errors.Wrap(err, "storcli getPhyDevs")
	}
	ret := make([]*MegaRaidPhyDev, len(pdList))
	for i, pd := range pdList {
		dev, err := pd.toMegaraidDev()
		if err != nil {
			return nil, errors.Wrapf(err, "toMegaraidDev %s", jsonutils.Marshal(pd))
		}
		ret[i] = dev
	}
	return ret, nil
}

type StorcliControllersInfo struct {
	Controllers []*StorcliControllerInfo `json:"Controllers"`
}

type StorcliControllerInfo struct {
	CommandStatus struct {
		CliVersion      string `json:"CLI Version"`
		OperatingSystem string `json:"Operating system"`
		Controller      int    `json:"Controller"`
		Status          string `json:"Status"`
	} `json:"Command Status"`
	ResponseData StorcliControllerData `json:"Response Data"`
}

type StorcliControllerData struct {
	Info []*StorcliPhysicalDrive `json:"Drive Information"`
}

type StorcliPhysicalDrive struct {
	EnclosureIdSlotNo   string `json:"EID:Slt"`
	DeviceId            string `json:"DID"`
	State               string `json:"State"`
	DriveGroup          string `json:"DG"`
	Size                string `json:"Size"`
	Interface           string `json:"Intf"`
	MediaType           string `json:"Med"`
	SelfEncryptiveDrive string `json:"SED"`
	ProtectionInfo      string `json:"PI"`
	SectorSize          string `json:"SeSz"`
	Model               string `json:"Model"`
	Spun                string `json:"Sp"`
	Type                string `json:"Type"`
}

func (d *StorcliPhysicalDrive) toMegaraidDev() (*MegaRaidPhyDev, error) {
	dev := NewMegaRaidPhyDev()
	if err := dev.fillByStorcliPD(d); err != nil {
		return nil, errors.Wrap(err, "fillByStorcliPD")
	}
	return dev, nil
}

// parseStorcliControllers parse command `storecli64 /c0/eall/sall show J` json output
func parseStorcliControllers(jsonStr string) (*StorcliControllersInfo, error) {
	obj, err := jsonutils.ParseString(jsonStr)
	if err != nil {
		return nil, err
	}
	info := new(StorcliControllersInfo)
	if err := obj.Unmarshal(info); err != nil {
		return nil, err
	}
	return info, nil
}

type StorcliLogicalVolumes struct {
	data jsonutils.JSONObject
}

func parseStorcliLVs(jsonStr string) (*StorcliLogicalVolumes, error) {
	obj, err := jsonutils.ParseString(jsonStr)
	if err != nil {
		return nil, err
	}
	return &StorcliLogicalVolumes{
		data: obj,
	}, nil
}

func (lvs *StorcliLogicalVolumes) controllersData() ([]jsonutils.JSONObject, error) {
	data, err := lvs.data.GetArray("Controllers")
	if err != nil {
		return nil, errors.Wrap(err, "Get Controllers")
	}
	return data, nil
}

func (lvs *StorcliLogicalVolumes) responseData(controller int) (*jsonutils.JSONDict, error) {
	cData, err := lvs.controllersData()
	if err != nil {
		return nil, errors.Wrap(err, "controllersData")
	}
	if len(cData) <= controller {
		return nil, jsonutils.ErrJsonDictKeyNotFound
	}
	data, err := cData[controller].Get("Response Data")
	if err != nil {
		return nil, errors.Wrap(err, "Get 'Response Data")
	}
	return data.(*jsonutils.JSONDict), nil
}

type StorcliLogicalVolumePD struct {
	// "EID:Slt": "8:0"
	EIDSlt string `json:"EID:Slt"`
	// "Size": "446.625GB"
	Size string `json:"Size"`
	// "Intf": "SATA"
	Intf string `json:"Intf"`
	// "MED": "SSD"
	Med string `json:"Med"`
	// "Model": "SAMSUNG MZ7LH480HAHQ-00005"
	Model string `json:"Model"`
}

type StorcliLogicalVolumeProperties struct {
	// "Strip Size": "256 KB"
	StripSize string `json:"Strip Size"`
	// "OS Drive Name": "/dev/sda"
	DeviceName string `json:"OS Drive Name"`
}

type StorcliLogicalVolume struct {
	// Name format "/c0/v0"
	Name string
	// "TYPE": "RAID0"
	Type string `json:"Type"`
	// "DG/VD": "0/0"
	DGVD string `json:"DG/VD"`
	// "Size": "446.625GB"
	Size string `json:"Size"`
	// PDs for VD
	PDs []*StorcliLogicalVolumePD
	// Properties
	Properties *StorcliLogicalVolumeProperties
	// index
	Index int
}

func fetchLvInfo(data *jsonutils.JSONDict, keyIdx string, vIdx int) (*StorcliLogicalVolume, error) {
	lvObj, err := data.Get(keyIdx)
	if err != nil {
		return nil, errors.Wrapf(err, "Get %s", keyIdx)
	}

	// parse PDs
	pdKey := fmt.Sprintf("PDs for VD %d", vIdx)
	if !data.Contains(pdKey) {
		return nil, nil
	}

	lvArr := make([]*StorcliLogicalVolume, 0)
	if err := lvObj.Unmarshal(&lvArr); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s to StorcliLogicalVolume", lvObj)
	}
	lv := lvArr[0]
	lv.Index = vIdx
	lv.Name = keyIdx
	pdObj, err := data.Get(pdKey)
	if err != nil {
		return nil, errors.Wrapf(err, "Get %s", pdKey)
	}
	pds := make([]*StorcliLogicalVolumePD, 0)
	if err := pdObj.Unmarshal(&pds); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s to PDs", pdObj)
	}
	lv.PDs = pds

	// parse Properties
	ppKey := fmt.Sprintf("VD%d Properties", vIdx)
	ppObj, err := data.Get(ppKey)
	if err != nil {
		return nil, errors.Wrapf(err, "Get %s", ppKey)
	}
	props := new(StorcliLogicalVolumeProperties)
	if err := ppObj.Unmarshal(props); err != nil {
		return nil, errors.Wrapf(err, "Unmarshal %s to Properties", ppObj)
	}
	lv.Properties = props

	return lv, nil
}

func (lvs *StorcliLogicalVolumes) GetLogicalVolumes(controller int) ([]*StorcliLogicalVolume, error) {
	data, err := lvs.responseData(controller)
	if err != nil {
		if errors.Cause(err) == jsonutils.ErrJsonDictKeyNotFound {
			return nil, nil
		}
		return nil, errors.Wrap(err, "responseData")
	}
	result := make([]*StorcliLogicalVolume, 0)
	dataMap, err := data.GetMap()
	if err != nil {
		return nil, errors.Wrap(err, "response data get map")
	}
	vdPrefix := fmt.Sprintf("/c%d/v", controller)
	for k := range dataMap {
		if strings.HasPrefix(k, vdPrefix) {
			// find a LV like cXvXXX
			vIdx, err := strconv.ParseInt(k[len(vdPrefix):], 10, 64)
			if err != nil {
				log.Errorf("key %s not a valid LV key: %s", k, err)
				continue
			}
			lv, err := fetchLvInfo(data, k, int(vIdx))
			if err != nil {
				log.Errorf("fetchLvInfo %s failed %s", k, err)
				continue
			}
			if lv != nil {
				result = append(result, lv)
			}
		}
	}
	return result, nil
}

func (lv *StorcliLogicalVolume) IsSSD() bool {
	isSSD := true
	for _, pd := range lv.PDs {
		if strings.ToLower(pd.Med) != "ssd" {
			return false
		}
	}
	return isSSD
}

func (lv *StorcliLogicalVolume) GetOSDevice() string {
	dev := lv.Properties.DeviceName
	if len(dev) == 0 {
		// try to guest device name
		dev = fmt.Sprintf("sd%c", 'a'+lv.Index)
	}
	return dev
}

func (lv *StorcliLogicalVolume) GetSysBlockRotationalPath() string {
	dev := filepath.Base(lv.GetOSDevice())
	return filepath.Join("/sys/block/", dev, "/queue/rotational")
}
