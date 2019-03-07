package storageman

import (
	"context"

	"yunion.io/x/onecloud/pkg/hostman/storageman/remotefile"
)

type IImageCache interface {
	GetPath() string
	GetName() string
	Load() bool
	Acquire(ctx context.Context, zone, srcUrl, format string) bool
	Release()
	Remove(ctx context.Context) error
	GetImageId() string

	GetDesc() *remotefile.SImageDesc
}
