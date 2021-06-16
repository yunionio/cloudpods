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

package uefi

import (
	"fmt"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	"yunion.io/x/onecloud/pkg/util/regutils2"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

const (
	// efibootmgr useage: https://github.com/rhboot/efibootmgr
	CMD_EFIBOOTMGR  = "/usr/sbin/efibootmgr"
	SUDO_EFIBOOTMGR = "sudo efibootmgr"

	MAC_KEYWORD = "MAC"
)

type BootMgr struct {
	// bootCurrent - the boot entry used to start the currently running system.
	bootCurrent string

	// bootOrder - the boot order as would appear in the boot manager.
	// The boot manager tries to boot the first active entry on this list.
	// If unsuccessful, it tries the next entry, and so on.
	bootOrder []string

	// bootNext - the boot entry which is scheduled to be run on next boot.
	// This superceeds BootOrder for one boot only, and is deleted by the
	// boot manager after first use.
	// This allows you to change the next boot behavior without changing BootOrder.
	bootNext string

	// timeout - the time in seconds between when the boot manager appears on the screen
	// until when it automatically chooses the startup value from BootNext or BootOrder.
	timeout int

	// entries - the boot entry parsed in map
	entries map[string]*BootEntry
}

type BootEntry struct {
	BootNum     string
	Description string
	IsActive    bool
}

func getEFIBootMgrCmd(sudo bool) string {
	if sudo {
		return SUDO_EFIBOOTMGR
	}
	return CMD_EFIBOOTMGR
}

func ParseEFIBootMGR(input string) (*BootMgr, error) {
	lines := strings.Split(input, "\n")

	mgr := &BootMgr{
		bootOrder: []string{},
		timeout:   -1,
		entries:   make(map[string]*BootEntry),
	}

	pf := func(ff func(string) bool) {
		for _, l := range lines {
			if ok := ff(l); ok {
				break
			}
		}
	}

	// parse BootCurrent
	pf(func(l string) bool {
		if current := parseEFIBootMGRBootCurrent(l); current != "" {
			mgr.bootCurrent = current
			return true
		}
		return false
	})

	// parse Timeout second
	pf(func(l string) bool {
		if timeout := parseEFIBootMGRTimeout(l); timeout != -1 {
			mgr.timeout = timeout
			return true
		}
		return false
	})

	// parse BootOrder
	pf(func(l string) bool {
		if order := parseEFIBootMGRBootOrder(l); len(order) != 0 {
			mgr.bootOrder = order
			return true
		}
		return false
	})

	// parse BootNext
	pf(func(l string) bool {
		if next := parseEFIBootMGRBootNext(l); next != "" {
			mgr.bootNext = next
			return true
		}
		return false
	})

	// parse entries
	pf(func(l string) bool {
		if entry := parseEFIBootMGREntry(l); entry != nil {
			mgr.entries[entry.BootNum] = entry
		}
		return false
	})

	// finally check
	if err := mgr.DataCheck(); err != nil {
		return nil, errors.Wrap(err, "Invalid efibootmgr parse")
	}

	return mgr, nil
}

func (m *BootMgr) DataCheck() error {
	if m.bootCurrent == "" {
		return errors.Error("BootCurrent is empty")
	}

	if len(m.bootOrder) == 0 {
		return errors.Error("BootOrder length is 0")
	}

	// check if BootOrder in entries
	for _, orderNum := range m.bootOrder {
		if _, ok := m.entries[orderNum]; !ok {
			return errors.Errorf("Not found BootOrder %s entry", orderNum)
		}
	}

	return nil
}

func parseEFIBootMGRBootCurrent(line string) string {
	prefix := "BootCurrent: "
	if strings.HasPrefix(line, prefix) {
		return strings.Split(line, prefix)[1]
	}
	return ""
}

func parseEFIBootMGRBootOrder(line string) []string {
	prefix := "BootOrder: "
	if !strings.HasPrefix(line, prefix) {
		return nil
	}
	orderStr := strings.Split(line, prefix)[1]
	return strings.Split(orderStr, ",")
}

func parseEFIBootMGRBootNext(line string) string {
	prefix := "BootNext: "
	if !strings.HasPrefix(line, prefix) {
		return ""
	}
	return strings.Split(line, prefix)[1]
}

func parseEFIBootMGRTimeout(line string) int {
	timeoutRegexp := `^Timeout: (?P<seconds>[0-9]{1,}) seconds`
	matches := regutils2.SubGroupMatch(timeoutRegexp, line)
	if len(matches) == 0 {
		return -1
	}
	secondStr := matches["seconds"]
	second, err := strconv.Atoi(secondStr)
	if err != nil {
		log.Errorf("parse %s seconds error: %v", secondStr, err)
		return -1
	}
	return second
}

func parseEFIBootMGREntry(line string) *BootEntry {
	entryRegexp := `^Boot(?P<num>[0-9a-zA-Z]{4})[^:]+?\s+(?P<description>.*)`
	matches := regutils2.SubGroupMatch(entryRegexp, line)
	if len(matches) == 0 {
		return nil
	}
	num, ok := matches["num"]
	if !ok {
		return nil
	}
	desc, ok := matches["description"]
	if !ok {
		return nil
	}
	isActive := false
	if strings.Contains(line, "* ") {
		isActive = true
	}
	return &BootEntry{
		BootNum:     num,
		Description: desc,
		IsActive:    isActive,
	}
}

func NewEFIBootMgrFromRemote(cli *ssh.Client, sudo bool) (*BootMgr, error) {
	return newEFIBootMgrFromRemote(cli, sudo, true)
}

func newEFIBootMgrFromRemote(cli *ssh.Client, sudo bool, verbose bool) (*BootMgr, error) {
	cmd := getEFIBootMgrCmd(sudo)
	if verbose {
		cmd = fmt.Sprintf("%s -v", cmd)
	}
	lines, err := cli.RawRun(cmd)
	if err != nil {
		return nil, errors.Wrapf(err, "Execute command: %s", cmd)
	}
	return ParseEFIBootMGR(strings.Join(lines, "\n"))
}

func (m *BootMgr) GetCommand(sudo bool) string {
	return getEFIBootMgrCmd(sudo)
}

func (m *BootMgr) GetBootCurrent() string {
	return m.bootCurrent
}

func (m *BootMgr) GetBootOrder() []string {
	return m.bootOrder
}

func (m *BootMgr) GetBootNext() string {
	return m.bootNext
}

func (m *BootMgr) GetTimeout() int {
	return m.timeout
}

func (m *BootMgr) GetBootEntry(num string) *BootEntry {
	return m.entries[num]
}

func (m *BootMgr) GetBootEntryByDesc(desc string) *BootEntry {
	for _, entry := range m.entries {
		if strings.Contains(entry.Description, desc) {
			return entry
		}
	}
	return nil
}

func (m *BootMgr) FindBootOrderPos(num string) int {
	return stringArraryFindItemPos(m.bootOrder, num)
}

func stringArraryFindItemPos(items []string, item string) int {
	for idx, elem := range items {
		if elem == item {
			return idx
		}
	}
	return -1
}

func stringArraryMove(items []string, item string, pos int) []string {
	origPos := stringArraryFindItemPos(items, item)
	if origPos == -1 {
		items = append(items, item)
		origPos = stringArraryFindItemPos(items, item)
	}

	for i := origPos; i != pos; {
		if i < pos {
			// from left to right
			tmp := items[i]
			items[i] = items[i+1]
			items[i+1] = tmp
			i++
		} else if i > pos {
			// from right to left
			tmp := items[i]
			items[i] = items[i-1]
			items[i-1] = tmp
			i--
		}
	}
	return items
}

func (m *BootMgr) MoveBootOrder(num string, pos int) *BootMgr {
	if entry := m.GetBootEntry(num); entry == nil {
		log.Warningf("Not found boot entry by %q", num)
		return m
	}
	m.bootOrder = stringArraryMove(m.bootOrder, num, pos)
	return m
}

func getSetBootOrderArgs(bootOrder []string) string {
	return strings.Join(bootOrder, ",")
}

func (m *BootMgr) GetSetBootOrderArgs() string {
	return getSetBootOrderArgs(m.bootOrder)
}

func RemoteIsUEFIBoot(cli *ssh.Client) (bool, error) {
	checkCmd := "test -d /sys/firmware/efi && echo is || echo not"
	lines, err := cli.Run(checkCmd)
	if err != nil {
		return false, err
	}
	for _, line := range lines {
		if strings.Contains(line, "is") {
			return true, nil
		}
	}
	return false, nil
}

func convertToEFIBootMgrInfo(info *BootMgr) (*types.EFIBootMgrInfo, error) {
	data := &types.EFIBootMgrInfo{
		PxeBootNum: info.GetBootCurrent(),
		BootOrder:  make([]*types.EFIBootEntry, len(info.GetBootOrder())),
	}
	for idx, orderNum := range info.GetBootOrder() {
		entry := info.GetBootEntry(orderNum)
		if entry == nil {
			return nil, errors.Errorf("Not found boot entry by %q", orderNum)
		}
		data.BootOrder[idx] = &types.EFIBootEntry{
			BootNum:     entry.BootNum,
			Description: entry.Description,
			IsActive:    entry.IsActive,
		}
	}
	return data, nil
}

func (mgr *BootMgr) ToEFIBootMgrInfo() (*types.EFIBootMgrInfo, error) {
	return convertToEFIBootMgrInfo(mgr)
}

func (mgr *BootMgr) sortEntryByKeyword(keyword string) *BootMgr {
	newOrder := []string{}
	oldOrder := []string{}
	for _, num := range mgr.bootOrder {
		entry := mgr.GetBootEntry(num)
		if strings.Contains(entry.Description, keyword) {
			newOrder = append(newOrder, num)
		} else {
			oldOrder = append(oldOrder, num)
		}
	}
	newOrder = append(newOrder, oldOrder...)
	mgr.bootOrder = newOrder
	return mgr
}

func RemoteSetCurrentBootAtFirst(cli *ssh.Client, mgr *BootMgr) error {
	curPos := mgr.FindBootOrderPos(mgr.GetBootCurrent())
	if curPos == -1 {
		return errors.Errorf("Not found BootCurrent position %q", mgr.GetBootCurrent())
	}
	// move to first
	mgr.MoveBootOrder(mgr.GetBootCurrent(), 0)
	cmd := fmt.Sprintf("%s -o %s", mgr.GetCommand(false), mgr.GetSetBootOrderArgs())
	_, err := cli.Run(cmd)
	return err
}

func RemoteSetBootOrder(cli *ssh.Client, order []string) error {
	cmd := fmt.Sprintf("%s -o %s", SUDO_EFIBOOTMGR, getSetBootOrderArgs(order))
	_, err := cli.RunWithTTY(cmd)
	return err
}

func RemoteSetBootOrderByInfo(cli *ssh.Client, entry *types.EFIBootEntry) (*BootMgr, error) {
	mgr, err := newEFIBootMgrFromRemote(cli, true, true)
	if err != nil {
		return nil, err
	}

	curEntry := mgr.GetBootEntryByDesc(entry.Description)
	if curEntry == nil {
		return nil, errors.Wrapf(err, "Not found remote boot entry by %q", entry.Description)
	}

	mgr = mgr.sortEntryByKeyword(curEntry.Description)
	return mgr, RemoteSetBootOrder(cli, mgr.GetBootOrder())
}

func RemoteTryToSetPXEBoot(cli *ssh.Client) error {
	mgr, err := newEFIBootMgrFromRemote(cli, true, true)
	if err != nil {
		return err
	}

	mgr = mgr.sortEntryByKeyword(MAC_KEYWORD)
	return RemoteSetBootOrder(cli, mgr.GetBootOrder())
}

func remoteISUEFIBootWrap(cli *ssh.Client, f func(*ssh.Client) error) error {
	isUEFI, err := RemoteIsUEFIBoot(cli)
	if err != nil {
		return errors.Wrap(err, "Check is UEFI boot")
	}
	if !isUEFI {
		return nil
	}
	return f(cli)
}

func RemoteTryRemoveOSBootEntry(hostCli *ssh.Client) error {
	return remoteISUEFIBootWrap(hostCli, remoteTryRemoveOSBootEntry)
}

func remoteTryRemoveOSBootEntry(hostCli *ssh.Client) error {
	mgr, err := newEFIBootMgrFromRemote(hostCli, false, true)
	if err != nil {
		return err
	}

	// TODO: find other ways to decide whether entry is OS boot
	osKeywords := []string{
		"linux",
		"centos",
		"ubuntu",
		"windows",
		"grub",
	}

	isOsEntry := func(desc string) bool {
		for _, key := range osKeywords {
			desc := strings.ToLower(desc)
			if strings.Contains(desc, key) {
				return true
			}
		}
		return false
	}

	for _, entry := range mgr.entries {
		if !isOsEntry(entry.Description) {
			continue
		}
		// delete entry and remove it from BootOrder
		cmd := fmt.Sprintf("%s -b %s -B", mgr.GetCommand(false), entry.BootNum)
		if _, err := hostCli.Run(cmd); err != nil {
			return errors.Wrapf(err, "remove boot entry: %s", entry.Description)
		}
	}
	return nil
}
