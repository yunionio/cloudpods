package lifecycle

import (
	"context"
	"fmt"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/util/pod"
)

var (
	drivers = make(map[apis.ContainerLifecyleHandlerType]ILifecycle)
)

func RegisterDriver(drv ILifecycle) {
	drivers[drv.GetType()] = drv
}

func GetDriver(typ apis.ContainerLifecyleHandlerType) ILifecycle {
	drv, ok := drivers[typ]
	if !ok {
		panic(fmt.Sprintf("not found driver by type %s", typ))
	}
	return drv
}

type ILifecycle interface {
	GetType() apis.ContainerLifecyleHandlerType
	Run(ctx context.Context, input *apis.ContainerLifecyleHandler, cri pod.CRI, id string) error
}
