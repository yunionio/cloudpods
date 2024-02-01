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

package isolated_device

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/util/regutils2"
)

type sUSBDevice struct {
	*SBaseDevice
	lsusbLine *sLsusbLine
}

// TODO: rename PCIDevice
func newUSBDevice(dev *PCIDevice, lsusbLine *sLsusbLine) *sUSBDevice {
	return &sUSBDevice{
		SBaseDevice: NewBaseDevice(dev, api.USB_TYPE),
		lsusbLine:   lsusbLine,
	}
}

func (dev *sUSBDevice) GetCPUCmd() string {
	return ""
}

func (dev *sUSBDevice) GetVGACmd() string {
	return ""
}

func (dev *sUSBDevice) CustomProbe(int) error {
	// do nothing
	return nil
}

func GetUSBDevId(vendorId, devId, bus, addr string) string {
	return fmt.Sprintf("dev_%s_%s-%s_%s", vendorId, devId, bus, addr)
}

func getUSBDevQemuOptions(vendorId, deviceId string, bus, addr string) (map[string]string, error) {
	// id := GetUSBDevId(vendorId, deviceId, bus, addr)
	busI, err := strconv.Atoi(bus)
	if err != nil {
		return nil, errors.Wrapf(err, "parse bus to int %q", bus)
	}
	addrI, err := strconv.Atoi(addr)
	if err != nil {
		return nil, errors.Wrapf(err, "parse addr to int %q", bus)
	}
	return map[string]string{
		// "id": id,
		// "bus":       "usb.0",
		"vendorid":  fmt.Sprintf("0x%s", vendorId),
		"productid": fmt.Sprintf("0x%s", deviceId),
		"hostbus":   fmt.Sprintf("%d", busI),
		"hostaddr":  fmt.Sprintf("%d", addrI),
	}, nil
}

func GetUSBDevQemuOptions(vendorDevId string, addr string) (map[string]string, error) {
	parts := strings.Split(vendorDevId, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid vendor_device_id %q", vendorDevId)
	}
	vendorId := parts[0]
	productId := parts[1]

	addrParts := strings.Split(addr, ":")
	if len(addrParts) != 2 {
		return nil, fmt.Errorf("invalid addr %q", addr)
	}
	hostBus := addrParts[0]
	hostAddr := addrParts[1]

	return getUSBDevQemuOptions(vendorId, productId, hostBus, hostAddr)
}

func (dev *sUSBDevice) GetKernelDriver() (string, error) {
	return "", nil
}

func (dev *sUSBDevice) GetQemuId() string {
	addrParts := strings.Split(dev.dev.Addr, ":")
	return GetUSBDevId(dev.dev.VendorId, dev.dev.DeviceId, addrParts[0], addrParts[1])
}

func (dev *sUSBDevice) GetPassthroughOptions() map[string]string {
	opts, _ := GetUSBDevQemuOptions(dev.dev.GetVendorDeviceId(), dev.dev.Addr)
	return opts
}

func (dev *sUSBDevice) GetPassthroughCmd(index int) string {
	opts, _ := GetUSBDevQemuOptions(dev.dev.GetVendorDeviceId(), dev.dev.Addr)
	optsStr := []string{}
	for k, v := range opts {
		optsStr = append(optsStr, fmt.Sprintf("%s=%s", k, v))
	}
	opt := fmt.Sprintf(" -device usb-host,%s", strings.Join(optsStr, ","))
	return opt
}

func (dev *sUSBDevice) GetHotPlugOptions(isolatedDev *desc.SGuestIsolatedDevice, guestDesc *desc.SGuestDesc) ([]*HotPlugOption, error) {
	opts, err := GetUSBDevQemuOptions(dev.dev.GetVendorDeviceId(), dev.dev.Addr)
	if err != nil {
		return nil, errors.Wrap(err, "GetUSBDevQemuOptions")
	}
	opts["id"] = isolatedDev.Usb.Id
	opts["bus"] = fmt.Sprintf("%s.0", guestDesc.Usb.Id)
	return []*HotPlugOption{
		{
			Device:  "usb-host",
			Options: opts,
		},
	}, nil
}

func (dev *sUSBDevice) GetHotUnplugOptions(*desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error) {
	return []*HotUnplugOption{
		{Id: dev.GetQemuId()},
	}, nil
}

func getPassthroughUSBs() ([]*sUSBDevice, error) {
	ret, err := bashOutput("lsusb")
	if err != nil {
		return nil, errors.Wrap(err, "execute lsusb")
	}
	lines := []string{}
	for _, l := range ret {
		if len(l) != 0 {
			lines = append(lines, l)
		}
	}

	devs, err := parseLsusb(lines)
	if err != nil {
		return nil, errors.Wrap(err, "parseLsusb")
	}

	treeRet, err := bashRawOutput("lsusb -t")
	if err != nil {
		return nil, errors.Wrap(err, "execute `lsusb -t`")
	}
	trees, err := parseLsusbTrees(treeRet)
	if err != nil {
		return nil, errors.Wrap(err, "parseLsusbTrees")
	}

	// fitler linux root hub
	retDev := make([]*sUSBDevice, 0)
	for _, dev := range devs {
		// REF: https://github.com/virt-manager/virt-manager/blob/0038d750c9056ddd63cb48b343e451f8db2746fa/virtinst/nodedev.py#L142
		if isUSBLinuxRootHub(dev.dev.VendorId, dev.dev.DeviceId) {
			continue
		}

		// check by trees
		isHubClass, err := isUSBHubClass(dev, trees)
		if err != nil {
			return nil, errors.Wrap(err, "check isUSBHubClass")
		}
		if isHubClass {
			continue
		}

		retDev = append(retDev, dev)
	}
	return retDev, nil
}

func isUSBLinuxRootHub(vendorId string, deviceId string) bool {
	if vendorId == "1d6b" && utils.IsInStringArray(deviceId, []string{"0001", "0002", "0003"}) {
		return true
	}
	return false
}

func isUSBHubClass(dev *sUSBDevice, trees *sLsusbTrees) (bool, error) {
	busNum, err := dev.lsusbLine.GetBusNumber()
	if err != nil {
		return false, errors.Wrapf(err, "GetBusNumber of dev %#v", dev.lsusbLine)
	}
	devNum, err := dev.lsusbLine.GetDeviceNumber()
	if err != nil {
		return false, errors.Wrapf(err, "GetDeviceNumber of dev %#v", dev.lsusbLine)
	}
	tree, ok := trees.GetBus(busNum)
	if !ok {
		return false, errors.Errorf("not found dev %#v by bus %d", dev.lsusbLine, busNum)
	}
	treeDev := tree.GetDevice(devNum)
	if treeDev == nil {
		return false, errors.Errorf("not found dev %#v by bus %d, dev %d", dev.lsusbLine, busNum, devNum)
	}

	return utils.IsInStringArray(treeDev.Class, []string{"root_hub", "Hub"}), nil
}

func parseLsusb(lines []string) ([]*sUSBDevice, error) {
	devs := make([]*sUSBDevice, 0)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		dev, err := parseLsusbLine(line)
		if err != nil {
			return nil, errors.Wrapf(err, "parseLsusbLine %q", line)
		}
		usbDev := newUSBDevice(dev.ToPCIDevice(), dev)
		devs = append(devs, usbDev)
	}
	return devs, nil
}

var (
	lsusbRegex = `^Bus (?P<bus_id>([0-9]{3})) Device (?P<device>([0-9]{3})): ID (?P<vendor_id>([0-9a-z]{4})):(?P<device_id>([0-9a-z]{4}))\s{0,1}(?P<name>(.*))`
)

type sLsusbLine struct {
	BusId    string `json:"bus_id"`
	Device   string `json:"device"`
	VendorId string `json:"vendor_id"`
	DeviceId string `json:"device_id"`
	Name     string `json:"name"`
}

func parseLsusbLine(line string) (*sLsusbLine, error) {
	ret := regutils2.SubGroupMatch(lsusbRegex, line)
	dev := new(sLsusbLine)
	if err := jsonutils.Marshal(ret).Unmarshal(dev); err != nil {
		return nil, err
	}
	return dev, nil
}

func (dev *sLsusbLine) ToPCIDevice() *PCIDevice {
	return &PCIDevice{
		Addr:      fmt.Sprintf("%s:%s", dev.BusId, dev.Device),
		VendorId:  dev.VendorId,
		DeviceId:  dev.DeviceId,
		ModelName: dev.Name,
	}
}

func (dev *sLsusbLine) GetBusNumber() (int, error) {
	return strconv.Atoi(dev.BusId)
}

func (dev *sLsusbLine) GetDeviceNumber() (int, error) {
	return strconv.Atoi(dev.Device)
}

type sLsusbTrees struct {
	Trees       map[int]*sLsusbTree
	sorted      bool
	sortedTrees sortLsusbTree
}

type sortLsusbTree []*sLsusbTree

func (t sortLsusbTree) Len() int {
	return len(t)
}

func (t sortLsusbTree) Swap(i, j int) {
	t[i], t[j] = t[j], t[i]
}

func (t sortLsusbTree) Less(i, j int) bool {
	it := t[i]
	jt := t[j]
	return it.Bus < jt.Bus
}

func newLsusbTrees() *sLsusbTrees {
	return &sLsusbTrees{
		Trees:       make(map[int]*sLsusbTree),
		sorted:      false,
		sortedTrees: sortLsusbTree([]*sLsusbTree{}),
	}
}

func (ts *sLsusbTrees) Add(bus int, tree *sLsusbTree) *sLsusbTrees {
	ts.Trees[bus] = tree
	return ts
}

func (ts *sLsusbTrees) sortTrees() *sLsusbTrees {
	if ts.sorted {
		return ts
	}
	for _, t := range ts.Trees {
		ts.sortedTrees = append(ts.sortedTrees, t)
	}
	sort.Sort(ts.sortedTrees)
	return ts
}

func (ts *sLsusbTrees) GetContent() string {
	ts.sortTrees()
	ret := []string{}
	for _, t := range ts.sortedTrees {
		ret = append(ret, t.GetContents()...)
	}
	return strings.Join(ret, "\n")
}

func (ts *sLsusbTrees) GetBus(bus int) (*sLsusbTree, bool) {
	t, ok := ts.Trees[bus]
	return t, ok
}

// parseLsusbTrees parses `lsusb -t` output
func parseLsusbTrees(lines []string) (*sLsusbTrees, error) {
	return _parseLsusbTrees(lines)
}

func _parseLsusbTrees(lines []string) (*sLsusbTrees, error) {
	trees := newLsusbTrees()
	var (
		prevTree *sLsusbTree
	)
	for idx, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		tree, err := newLsusbTreeByLine(line)
		if err != nil {
			return nil, errors.Wrapf(err, "by line %q, index %d", line, idx)
		}
		if tree.IsRootBus {
			tree.parentNode = nil
			trees.Add(tree.Bus, tree)
			prevTree = tree
		} else {
			if len(tree.LinePrefix) > len(prevTree.LinePrefix) {
				// child node should be added to previous node
				tree.parentNode = prevTree
				tree.parentNode.AddNode(tree)
			} else if len(tree.LinePrefix) == len(prevTree.LinePrefix) {
				// sibling node should be added to parent
				prevTree.parentNode.AddNode(tree)
			} else if len(tree.LinePrefix) < len(prevTree.LinePrefix) {
				// find current node's sibling node and added to it's parent
				parent := prevTree.FindParentByTree(tree)
				if parent == nil {
					return nil, errors.Errorf("can't found parent by tree %s, current line %q, prevTree %s", jsonutils.Marshal(tree), line, jsonutils.Marshal(prevTree))
				}
				parent.AddNode(tree)
			}
			prevTree = tree
		}
	}
	return trees, nil
}

const (
	busRootPrefix = "/:  Bus"
)

var (
	lsusbTreeRootBusRegex     = `(?P<prefix>(.*))Bus (?P<bus_id>([0-9]{2}))\.`
	lsusbTreeBusSuffixRegex   = `Port (?P<port_id>([0-9]{1,2})): Dev (?P<device>([0-9]{1,2})), Class=(?P<class>(.*)), Driver=(?P<driver>(.*)),\s{0,1}(?P<speed>(.*))`
	lsusbTreeSuffixRegex      = `Port (?P<port_id>([0-9]{1,2})): Dev (?P<device>([0-9]{1,2})), If (?P<interface>([0-9]{1,2})), Class=(?P<class>(.*)), Driver=(?P<driver>(.*)),\s{0,1}(?P<speed>(.*))`
	lsusbTreeRootBusLineRegex = lsusbTreeRootBusRegex + lsusbTreeBusSuffixRegex
	lsusbTreeLineRegex        = `(?P<prefix>(.*))` + lsusbTreeSuffixRegex
)

type sLsusbTree struct {
	parentNode *sLsusbTree

	IsRootBus  bool   `json:"is_root_bus"`
	LinePrefix string `json:"line_prefix"`
	Bus        int    `json:"bus"`
	Port       int    `json:"port"`
	Dev        int    `json:"dev"`
	// If maybe nil
	If      int           `json:"if"`
	Class   string        `json:"class"`
	Driver  string        `json:"driver"`
	Content string        `json:"content`
	Nodes   []*sLsusbTree `json:"nodes"`
}

func newLsusbTreeByLine(line string) (*sLsusbTree, error) {
	var (
		isRootBus = false
		regExp    = lsusbTreeLineRegex
	)
	if strings.HasPrefix(line, busRootPrefix) {
		isRootBus = true
	}
	if isRootBus {
		regExp = lsusbTreeRootBusLineRegex
	}
	ret := regutils2.SubGroupMatch(regExp, line)
	linePrefix := ret["prefix"]
	if linePrefix == "" {
		return nil, errors.Errorf("not found prefix of line %q", line)
	}
	t := &sLsusbTree{
		IsRootBus:  isRootBus,
		Content:    line,
		LinePrefix: linePrefix,
		Nodes:      make([]*sLsusbTree, 0),
	}

	if isRootBus {
		// parse bus
		busIdStr, ok := ret["bus_id"]
		if !ok {
			return nil, errors.Errorf("not found 'Bus' in %q", line)
		}
		busId, err := strconv.Atoi(busIdStr)
		if err != nil {
			return nil, errors.Errorf("invalid Bus string %q", busIdStr)
		}
		t.Bus = busId
	}

	// parse port
	portIdStr, ok := ret["port_id"]
	if !ok {
		return nil, errors.Errorf("not found 'Port' in %q", line)
	}
	portId, err := strconv.Atoi(portIdStr)
	if err != nil {
		return nil, errors.Errorf("invalid Port string %q", portIdStr)
	}
	t.Port = portId

	// parse dev
	devStr, ok := ret["device"]
	if !ok {
		return nil, errors.Errorf("not found 'Dev' in %q", line)
	}
	dev, err := strconv.Atoi(devStr)
	if err != nil {
		return nil, errors.Errorf("invalid Dev string %q", devStr)
	}
	t.Dev = dev

	// parse if when not root bus
	if !isRootBus {
		ifStr, ok := ret["interface"]
		if !ok {
			return nil, errors.Errorf("not found 'If' in %q", line)
		}
		ifN, err := strconv.Atoi(ifStr)
		if err != nil {
			return nil, errors.Errorf("invalid ifStr string %q", ifStr)
		}
		t.If = ifN
	}

	// parse class
	class, ok := ret["class"]
	if !ok {
		return nil, errors.Errorf("not found 'Class' in %q", line)
	}
	t.Class = class

	// parse driver
	driver := ret["driver"]
	t.Driver = driver

	return t, nil
}

func (t *sLsusbTree) AddNode(child *sLsusbTree) *sLsusbTree {
	child.Bus = t.Bus
	child.parentNode = t
	t.Nodes = append(t.Nodes, child)
	return t
}

func (t *sLsusbTree) FindParentByTree(it *sLsusbTree) *sLsusbTree {
	tl := len(t.LinePrefix)
	itl := len(it.LinePrefix)
	if tl == itl {
		return t.parentNode
	} else if tl > itl {
		return t
	} else {
		return t.parentNode.FindParentByTree(it)
	}
}

func (t *sLsusbTree) GetContents() []string {
	ret := []string{t.Content}
	for _, n := range t.Nodes {
		ret = append(ret, n.GetContents()...)
	}
	return ret
}

func (t *sLsusbTree) GetDevice(devNum int) *sLsusbTree {
	// should check self firstly
	if t.Dev == devNum {
		return t
	}
	// then check children
	for _, node := range t.Nodes {
		dev := node.GetDevice(devNum)
		if dev != nil {
			return dev
		}
	}
	return nil
}
