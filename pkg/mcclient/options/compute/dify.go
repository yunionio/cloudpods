package compute

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

type DifyCreateOptions struct {
	// Below are PodCreateOptions without ContainerCreateCommonOptions and AutoStart
	NAME string `help:"Name of server pod" json:"-"`
	ServerCreateCommonConfig
	MEM              string `help:"Memory size MB" metavar:"MEM" json:"-"`
	VcpuCount        int    `help:"#CPU cores of VM server, default 1" default:"1" metavar:"<SERVER_CPU_COUNT>" json:"vcpu_count" token:"ncpu"`
	AllowDelete      *bool  `help:"Unlock server to allow deleting" json:"-"`
	Arch             string `help:"image arch" choices:"aarch64|x86_64"`
	ShutdownBehavior string `help:"Behavior after VM server shutdown" metavar:"<SHUTDOWN_BEHAVIOR>" choices:"stop|terminate|stop_release_gpu"`
	PodUid           int64  `help:"UID of pod" default:"0"`
	PodGid           int64  `help:"GID of pod" default:"0"`

	// Below are dify custom options
}

func (o *DifyCreateOptions) Params() (jsonutils.JSONObject, error) {
	// use PodCreateOptions to param
	podCreatOpt := &PodCreateOptions{
		NAME:                     o.NAME,
		MEM:                      o.MEM,
		VcpuCount:                o.VcpuCount,
		AllowDelete:              o.AllowDelete,
		Arch:                     o.Arch,
		AutoStart:                true,
		ShutdownBehavior:         o.ShutdownBehavior,
		PodUid:                   o.PodUid,
		PodGid:                   o.PodGid,
		ServerCreateCommonConfig: o.ServerCreateCommonConfig,
	}
	input, err := podCreatOpt.Params()
	if err != nil {
		return nil, err
	}

	return jsonutils.Marshal(input), nil
}

type DifyListOptions struct {
	options.BaseListOptions
	GuestId string `json:"guest_id" help:"guest(pod) id or name"`
}

func (o *DifyListOptions) Params() (jsonutils.JSONObject, error) {
	return options.ListStructToParams(o)
}

type DifyIdOptions struct {
	ID string `help:"ID or name of the dify" json:"-"`
}

func (o *DifyIdOptions) GetId() string {
	return o.ID
}

func (o *DifyIdOptions) Params() (jsonutils.JSONObject, error) {
	return nil, nil
}

type DifyShowOptions struct {
	DifyIdOptions
}
