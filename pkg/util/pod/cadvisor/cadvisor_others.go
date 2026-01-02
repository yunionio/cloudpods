//go:build !linux
// +build !linux

package cadvisor

import (
	"yunion.io/x/pkg/errors"
)

func New(imageFsInfoProvider ImageFsInfoProvider, rootPath string, cgroupRoots []string) (Interface, error) {
	return nil, errors.ErrNotImplemented
}
