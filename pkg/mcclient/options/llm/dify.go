package llm

import (
	"reflect"
	"regexp"
	"strings"

	"yunion.io/x/jsonutils"
	llmapi "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

type DifyCustomizedEnvOptions struct {
	ConsoleApiUrl string `help:"The backend URL of the console API, used to concatenate the authorization callback."`
	ConsoleWebUrl string `help:"The front-end URL of the console web,used to concatenate some front-end addresses and for CORS configuration use."`
	ServiceApiUrl string `help:"Service API Url,used to display Service API Base Url to the front-end."`
	AppApiUrl     string `help:"WebApp API backend Url,used to declare the back-end URL for the front-end API."`
	AppWebUrl     string `help:"WebApp Url,used to display WebAPP API Base Url to the front-end."`
}

func (o *DifyCustomizedEnvOptions) getDifyCustomizedEnvs() []*llmapi.DifyCustomizedEnv {
	var envs []*llmapi.DifyCustomizedEnv
	val := reflect.ValueOf(o).Elem()

	var snakeCaseConverter = regexp.MustCompile("([a-z0-9])([A-Z])")

	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)

		if valueField.Kind() == reflect.String {
			fieldValue := valueField.String()
			if fieldValue != "" {
				envs = append(envs, &llmapi.DifyCustomizedEnv{
					Key:   strings.ToUpper(snakeCaseConverter.ReplaceAllString(typeField.Name, "${1}_${2}")),
					Value: fieldValue,
				})
			}
		}
	}

	return envs
}

type DifyCustomized struct {
	DifyCustomizedEnvOptions
	REGISTRY string `help:"Registry of the image, need container such images(default in docker.io): postgres:15-alpine, redis:6-alpine, nginx:latest, langgenius/dify-api:1.7.2, langgenius/dify-plugin-daemon:0.2.0-local, langgenius/dify-web:1.7.2, langgenius/dify-sandbox:0.2.12, ubuntu/squid:latest, semitechnologies/weaviate:1.19.0"`
}

func (o *DifyCustomized) getDifyCustomized() *llmapi.DifyCustomized {
	return &llmapi.DifyCustomized{
		CustomizedEnvs: o.getDifyCustomizedEnvs(),
		Registry:       o.REGISTRY,
	}
}

type DifyCreateOptions struct {
	// Below are PodCreateOptions without ContainerCreateCommonOptions and AutoStart
	NAME string `help:"Name of server pod" json:"-"`
	compute.ServerCreateCommonConfig
	MEM              string `help:"Memory size MB" metavar:"MEM" json:"-"`
	VcpuCount        int    `help:"#CPU cores of VM server, default 1" default:"1" metavar:"<SERVER_CPU_COUNT>" json:"vcpu_count" token:"ncpu"`
	AllowDelete      *bool  `help:"Unlock server to allow deleting" json:"-"`
	Arch             string `help:"image arch" choices:"aarch64|x86_64"`
	ShutdownBehavior string `help:"Behavior after VM server shutdown" metavar:"<SHUTDOWN_BEHAVIOR>" choices:"stop|terminate|stop_release_gpu"`
	PodUid           int64  `help:"UID of pod" default:"0"`
	PodGid           int64  `help:"GID of pod" default:"0"`

	// Below are dify custom options
	DifyCustomized
}

func (o *DifyCreateOptions) Params() (jsonutils.JSONObject, error) {
	input := &llmapi.DifyCreateInput{}

	// use PodCreateOptions to param
	podCreatOpt := &compute.PodCreateOptions{
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
	serverInput, err := podCreatOpt.Params()
	if err != nil {
		return nil, err
	}
	input.ServerCreateInput = *serverInput

	// get DifyCustomizedEnvOptions
	input.DifyCustomized = *o.DifyCustomized.getDifyCustomized()

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
