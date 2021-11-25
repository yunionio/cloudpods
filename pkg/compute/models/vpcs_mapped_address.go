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

package models

import (
	"context"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

const (
	errMappedIpExhausted = errors.Error("mapped ip exhausted")

	LOCK_CLASS_guestnetworks_mapped_addr = "guestnetworks-mapped-addr"
	LOCK_OBJ_guestnetworks_mapped_addr   = "the-addr"

	LOCK_CLASS_hosts_mapped_addr = "hosts-mapped-addr"
	LOCK_OBJ_hosts_mapped_addr   = "the-addr"
)

func (man *SGuestnetworkManager) lockAllocMappedAddr(ctx context.Context) {
	lockman.LockRawObject(ctx, LOCK_CLASS_guestnetworks_mapped_addr, LOCK_OBJ_guestnetworks_mapped_addr)
}

func (man *SGuestnetworkManager) unlockAllocMappedAddr(ctx context.Context) {
	lockman.ReleaseRawObject(ctx, LOCK_CLASS_guestnetworks_mapped_addr, LOCK_OBJ_guestnetworks_mapped_addr)
}

func (man *SGuestnetworkManager) allocMappedIpAddr(ctx context.Context) (string, error) {
	var (
		used []string
		ip   string
	)

	q := man.Query("mapped_ip_addr").IsNotEmpty("mapped_ip_addr")
	rows, err := q.Rows()
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&ip); err != nil {
			return "", errors.Wrap(err, "scan guest mapped ip")
		}
		used = append(used, ip)
	}

	sip := api.VpcMappedIPStart()
	eip := api.VpcMappedIPEnd()
	for i := eip; i >= sip; i-- {
		s := i.String()
		if !utils.IsInStringArray(s, used) {
			return s, nil
		}
	}
	return "", errors.Wrap(errMappedIpExhausted, "guests")
}

func (man *SHostManager) lockAllocOvnMappedIpAddr(ctx context.Context) {
	lockman.LockRawObject(ctx, LOCK_CLASS_hosts_mapped_addr, LOCK_OBJ_hosts_mapped_addr)
}

func (man *SHostManager) unlockAllocOvnMappedIpAddr(ctx context.Context) {
	lockman.ReleaseRawObject(ctx, LOCK_CLASS_hosts_mapped_addr, LOCK_OBJ_hosts_mapped_addr)
}

func (man *SHostManager) allocOvnMappedIpAddr(ctx context.Context) (string, error) {
	var (
		used []string
		ip   string
	)

	q := man.Query("ovn_mapped_ip_addr").IsNotEmpty("ovn_mapped_ip_addr")
	rows, err := q.Rows()
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		if err := rows.Scan(&ip); err != nil {
			return "", errors.Wrap(err, "scan host mapped ip")
		}
		used = append(used, ip)
	}

	sip := api.VpcMappedHostIPStart()
	eip := api.VpcMappedHostIPEnd()
	for i := sip; i <= eip; i++ {
		s := i.String()
		if !utils.IsInStringArray(s, used) {
			return s, nil
		}
	}
	return "", errors.Wrap(errMappedIpExhausted, "hosts")
}
