package models

import (
	"context"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
)

const (
	errMappedIpExhausted = errors.Error("mapped ip exhaused")
)

func (man *SGuestnetworkManager) allocMappedIpAddr(ctx context.Context) (string, error) {
	const (
		LOCK_CLASS = "guestnetworks-mapped-addr"
		LOCK_OBJ   = "the-addr"
	)
	lockman.LockRawObject(ctx, LOCK_CLASS, LOCK_OBJ)
	defer lockman.ReleaseRawObject(ctx, LOCK_CLASS, LOCK_OBJ)
	return man.allocMappedIpAddr_(ctx)
}

func (man *SGuestnetworkManager) allocMappedIpAddr_(ctx context.Context) (string, error) {
	var (
		used []string
		ip   string
	)

	q := man.Query("mapped_ip_addr").IsNotEmpty("mapped_ip_addr")
	rows, err := q.Rows()
	if err != nil {
		return "", err
	}
	for rows.Next() {
		if err := rows.Scan(&ip); err != nil {
			return "", errors.Wrap(err, "scan guest mapped ip")
		}
		used = append(used, ip)
	}

	sip := api.VpcMappedIPStart()
	eip := api.VpcMappedIPEnd()
	for i := sip; i <= eip; i++ {
		s := i.String()
		if !utils.IsInStringArray(s, used) {
			return s, nil
		}
	}
	return "", errors.Wrap(errMappedIpExhausted, "guests")
}

func (man *SHostManager) allocOvnMappedIpAddr(ctx context.Context) (string, error) {
	const (
		LOCK_CLASS = "hosts-vpc-mapped-addr"
		LOCK_OBJ   = "the-addr"
	)
	lockman.LockRawObject(ctx, LOCK_CLASS, LOCK_OBJ)
	defer lockman.ReleaseRawObject(ctx, LOCK_CLASS, LOCK_OBJ)
	return man.allocOvnMappedIpAddr_(ctx)
}

func (man *SHostManager) allocOvnMappedIpAddr_(ctx context.Context) (string, error) {
	var (
		used []string
		ip   string
	)

	q := man.Query("ovn_mapped_ip_addr").IsNotEmpty("ovn_mapped_ip_addr")
	rows, err := q.Rows()
	if err != nil {
		return "", err
	}
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
