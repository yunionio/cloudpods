package cadvisor

import (
	"github.com/google/cadvisor/events"
	cadvisorapi "github.com/google/cadvisor/info/v1"
	cadvisorapiv2 "github.com/google/cadvisor/info/v2"
)

type Interface interface {
	Start() error
	ContainerInfo(name string, req *cadvisorapi.ContainerInfoRequest) (*cadvisorapi.ContainerInfo, error)
	ContainerInfoV2(name string, options cadvisorapiv2.RequestOptions) (map[string]cadvisorapiv2.ContainerInfo, error)
	MachineInfo() (*cadvisorapi.MachineInfo, error)
	VersionInfo() (*cadvisorapi.VersionInfo, error)

	// Returns usage information about the filesystem holding container images.
	ImagesFsInfo() (cadvisorapiv2.FsInfo, error)

	// Returns usage information about the root filesystem.
	RootFsInfo() (cadvisorapiv2.FsInfo, error)

	// Get events streamed through passedChannel that fit the request.
	WatchEvents(request *events.Request) (*events.EventChannel, error)

	// Get filesystem information for the filesystem that contains the given file.
	GetDirFsInfo(path string) (cadvisorapiv2.FsInfo, error)
}

// ImageFsInfoProvider informs cAdvisor how to find imagefs for container images.
type ImageFsInfoProvider interface {
	// ImageFsInfoLabel returns the label cAdvisor should use to find the filesystem holding container images.
	ImageFsInfoLabel() (string, error)
}
