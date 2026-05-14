//go:build !linux
// +build !linux

package snapshot_service

import (
	"yunion.io/x/pkg/errors"
)

func StartService(guestMan IGuestManager, root string) error {
	return errors.ErrNotImplemented
}
