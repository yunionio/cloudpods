package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/rand"

	api "yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

const (
	NV_DEV_UVM       = "/dev/nvidia-uvm"
	NV_DEV_UVM_TOOLS = "/dev/nvidia-uvm-tools"

	DEV_DRI = "/dev/dri/"

	NV_DEV_CAP1    = "/dev/nvidia-caps/nvidia-cap1"
	NV_DEV_CAP2    = "/dev/nvidia-caps/nvidia-cap2"
	NV_DEV_CTL     = "/dev/nvidiactl"
	NV_DEV_MODESET = "/dev/nvidia-modeset"
	DEV_UINPUT     = "/dev/uinput"
	DEV_UHID       = "/dev/uhid"

	VOL_DEV      = "/dev"
	VOL_RUN_UDEV = "/run/udev"

	CGROUP_RULE_13  = "c 13:* rmw"
	CGROUP_RULE_244 = "c 244:* rmw"
)

var (
	authUrl  string
	user     string
	password string
	region   string

	podNet     string
	podIP      string
	podName    string
	diskSizeGB int
	ncpu       int
	mem        int
	basePort   int
	accessPort int

	wolfImage   string
	steamImage  string
	externalIP  string
	enableLxcfs bool

	gpu      string
	gpuModel string
	gpuType  string
	gpuEnvId string
	// renderNode           string
	overlay              string
	alwaysMountDriverVol bool

	devs string

	devsList []string

	wolfAllGpu       bool
	mounts           string
	mountList        []string
	appMounts        string
	appMountList     []string
	steamNoBigScreen bool
)

func init() {
	flag.StringVar(&authUrl, "auth-url", "", "auth url")
	flag.StringVar(&user, "user", "", "user")
	flag.StringVar(&password, "password", "", "password")
	flag.StringVar(&region, "region", "", "region")
	flag.StringVar(&podNet, "net", "", "pod net")
	flag.StringVar(&podIP, "ip", "", "pod ip")
	flag.StringVar(&podName, "name", "steam", "pod name")
	flag.IntVar(&ncpu, "ncpu", 8, "cpu count")
	flag.IntVar(&mem, "mem", 16, "memory in GB")
	flag.IntVar(&diskSizeGB, "disk-size", 10, "disk size in GB")
	flag.IntVar(&accessPort, "port", 20105, "moonlight access http port")
	// - registry.cn-beijing.aliyuncs.com/zexi/wolf:hook-0408.0: stable version
	flag.StringVar(&wolfImage, "wolf-image", "registry.cn-beijing.aliyuncs.com/zexi/wolf:patch-191-0420.0", "wolf image")
	flag.StringVar(&steamImage, "steam-image", "registry.cn-beijing.aliyuncs.com/zexi/steam:custom.4", "steam image")
	flag.StringVar(&externalIP, "eip", "", "external ip")
	flag.BoolVar(&enableLxcfs, "lxcfs", false, "enable lxcfs")
	flag.BoolVar(&alwaysMountDriverVol, "mount-driver-vol", false, "always mount driver volume")
	flag.StringVar(&gpu, "gpu", "", "gpu")
	flag.StringVar(&gpuEnvId, "gpu-env-id", "", "gpu env id")
	flag.StringVar(&gpuModel, "gpu-model", "", "gpu model")
	flag.StringVar(&gpuType, "gpu-type", "", "gpu type")
	// flag.StringVar(&renderNode, "render-node", "/dev/dri/renderD128", "render node")
	flag.StringVar(&overlay, "overlay", "", "overlay")
	flag.StringVar(&devs, "devs", "", "devs")
	flag.StringVar(&mounts, "mounts", "", "mounts")
	flag.StringVar(&appMounts, "app-mounts", "", "app mounts")
	flag.BoolVar(&wolfAllGpu, "wolf-all-gpu", false, "wolf all gpu")
	flag.BoolVar(&steamNoBigScreen, "steam-no-big-screen", false, "steam no big screen")
	flag.Parse()

	basePort = accessPort - 5
	initAuthInfo()
	log.Infof("Connecting to %s as %s", authUrl, user)

	devsList = strings.Split(devs, ",")
	mountList = strings.Split(mounts, ",")
	appMountList = strings.Split(appMounts, ",")
}

func initAuthInfo() {
	if authUrl == "" {
		authUrl = os.Getenv("OS_AUTH_URL")
	}
	if user == "" {
		user = os.Getenv("OS_USERNAME")
	}
	if password == "" {
		password = os.Getenv("OS_PASSWORD")
	}
	if region == "" {
		region = os.Getenv("OS_REGION_NAME")
	}
}

func NewHostDev(path string) *compute.ContainerDevice {
	return &compute.ContainerDevice{
		Type: api.CONTAINER_DEVICE_TYPE_HOST,
		Host: &compute.ContainerHostDevice{
			HostPath:      path,
			ContainerPath: path,
			Permissions:   "rwm",
		},
	}
}

func NewEnv(key, val string) *api.ContainerKeyValue {
	return &api.ContainerKeyValue{
		Key:   key,
		Value: val,
	}
}

func getMounts(mountList []string) []*api.ContainerVolumeMount {
	ret := make([]*api.ContainerVolumeMount, len(mountList))
	for i, m := range mountList {
		parts := strings.Split(m, ":")
		if len(parts) != 2 {
			log.Fatalf("Invalid mount spec: %s", m)
		}
		uniqName := fmt.Sprintf("%s_%s", m, rand.String(2))
		ret[i] = &api.ContainerVolumeMount{
			UniqueName: uniqName,
			Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
			MountPath:  parts[1],
			HostPath: &api.ContainerVolumeMountHostPath{
				Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE,
				Path: parts[0],
			},
		}
	}
	return ret
}

func getTmpSocketsHostPath(name string) string {
	return fmt.Sprintf("/tmp/%s/sockets", name)
}

func NewPulseAudioContainer(podName string, enableLxcfs bool) *compute.PodContainerCreateInput {
	return &compute.PodContainerCreateInput{
		ContainerSpec: compute.ContainerSpec{
			ContainerSpec: api.ContainerSpec{
				Command:         []string{"/entrypoint.sh"},
				Image:           "registry.cn-beijing.aliyuncs.com/zexi/pulseaudio:master",
				ImagePullPolicy: api.ImagePullPolicyAlways,
				EnableLxcfs:     enableLxcfs,
				Envs: []*api.ContainerKeyValue{
					NewEnv("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"),
					NewEnv("UNAME", "retro"),
					NewEnv("UID", "1000"),
					NewEnv("GID", "1000"),
					NewEnv("XDG_RUNTIME_DIR", "/tmp/pulse"),
					NewEnv("UMASK", "000"),
					NewEnv("HOME", "/root"),
				},
			},
			VolumeMounts: []*api.ContainerVolumeMount{
				{
					UniqueName: "tmp-pulse-audio",
					Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
					MountPath:  "/tmp/pulse",
					HostPath: &api.ContainerVolumeMountHostPath{
						Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
						Path: getTmpSocketsHostPath(podName),
					},
				},
			},
		},
	}
}

func getNvidiaManualBaseDevs() []*compute.ContainerDevice {
	return []*compute.ContainerDevice{
		NewHostDev(NV_DEV_UVM),
		NewHostDev(NV_DEV_UVM_TOOLS),
		NewHostDev(NV_DEV_CAP1),
		NewHostDev(NV_DEV_CAP2),
		NewHostDev(NV_DEV_CTL),
		NewHostDev(NV_DEV_MODESET),
	}
}

func getNvidiaNvDevs(idx int) []*compute.ContainerDevice {
	baseDevs := getNvidiaManualBaseDevs()
	dev := NewHostDev(fmt.Sprintf("/dev/nvidia%d", idx))
	baseDevs = append(baseDevs, dev)
	return baseDevs
}

func getNvidiaAppDevs(idx int) []*compute.ContainerDevice {
	devs := getNvidiaNvDevs(idx)
	driDev := NewHostDev(fmt.Sprintf("/dev/dri/card%d", idx))
	renderDev := NewHostDev(fmt.Sprintf("/dev/dri/renderD%d", idx+128))
	devs = append(devs, driDev, renderDev)
	return devs
}

func NewWolfContainer(i CreateInput) *compute.PodContainerCreateInput {
	zero := 0
	eip := i.IP
	if i.ExternalIP != "" {
		eip = i.ExternalIP
	}
	envs := []*api.ContainerKeyValue{
		// NewEnv("WOLF_LOG_LEVEL", "DEBUG"),
		NewEnv("WOLF_BASE_PORT", fmt.Sprintf("%d", i.BasePort)),
		NewEnv("WOLF_EXTERNAL_IP", eip),
		NewEnv("HOST_APPS_STATE_FOLDER", "/etc/wolf"),
		NewEnv("XDG_RUNTIME_DIR", "/tmp/sockets"),
		// NewEnv("WOLF_RENDER_NODE", i.RenderNode),
	}
	envs = append(envs, getPortEnvs(i.BasePort)...)
	if i.GPU == "" || i.WolfAllGpu {
		envs = append(envs, NewEnv("NVIDIA_DRIVER_VOLUME_NAME", "nvidia-driver-vol"))
	}
	if i.WolfAllGpu {
		envs = append(envs,
			NewEnv("NVIDIA_VISIBLE_DEVICES", "all"),
			NewEnv("NVIDIA_DRIVER_CAPABILITIES", "all"))
	}

	devs := []*compute.ContainerDevice{
		NewHostDev(DEV_UINPUT),
		NewHostDev(DEV_UHID),
	}
	if i.GPU == "" && i.GPUModel == "" && i.GPUType == "" {
		if !i.WolfAllGpu {
			devs = append(devs, NewHostDev(DEV_DRI))
			devs = append(devs, getNvidiaNvDevs(0)...)
		}
	} else {
		id0 := 0
		if !i.WolfAllGpu {
			devs = append(devs, &compute.ContainerDevice{
				Type: api.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE,
				IsolatedDevice: &compute.ContainerIsolatedDevice{
					Index: &id0,
				},
			})
		} else {
			devs = append(devs, &compute.ContainerDevice{
				Type: api.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE,
				IsolatedDevice: &compute.ContainerIsolatedDevice{
					Index: &id0,
					OnlyEnv: []*api.ContainerIsolatedDeviceOnlyEnv{
						{
							Key:            "WOLF_RENDER_NODE",
							FromRenderPath: true,
						},
					},
				},
			})
		}
	}
	vms := []*api.ContainerVolumeMount{
		/*{
			UniqueName: "etc-wolf",
			Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
			MountPath:  "/etc/wolf",
			HostPath: &api.ContainerVolumeMountHostPath{
				Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
				Path: "/etc/wolf",
			},
			Propagation: api.MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL,
		},*/
		{
			UniqueName: "wolf-data",
			Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
			MountPath:  "/etc/wolf",
			Disk: &api.ContainerVolumeMountDisk{
				Index:        &zero,
				SubDirectory: "wolf",
			},
		},
		{
			UniqueName: "run-udev",
			Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
			MountPath:  VOL_RUN_UDEV,
			HostPath: &api.ContainerVolumeMountHostPath{
				Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
				Path: VOL_RUN_UDEV,
			},
			//Propagation: api.MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL,
		},
		{
			UniqueName: "dev",
			Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
			MountPath:  VOL_DEV,
			HostPath: &api.ContainerVolumeMountHostPath{
				Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
				Path: VOL_DEV,
			},
			// WARN: 这里不能用 bidirectional mount
			// Propagation: api.MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL,
		},
		{
			UniqueName: "tmp-sockets",
			Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
			MountPath:  "/tmp/sockets",
			HostPath: &api.ContainerVolumeMountHostPath{
				Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
				//Path: "/tmp/sockets",
				Path: getTmpSocketsHostPath(i.Name),
			},
			Propagation: api.MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL,
		},
		// {
		// 	UniqueName: "tmp-pulse-audio",
		// 	Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
		// 	MountPath:  "/tmp/sockets/pulse-socket",
		// 	HostPath: &api.ContainerVolumeMountHostPath{
		// 		Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE,
		// 		Path: "/tmp/sockets/pulse-socket",
		// 	},
		// 	// Propagation: api.MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL,
		// },
		{
			UniqueName: "docker-socket",
			Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
			MountPath:  "/var/run/docker.sock",
			HostPath: &api.ContainerVolumeMountHostPath{
				Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE,
				Path: "/var/run/docker.sock",
			},
		},
	}
	vms = append(vms, getMounts(i.MountList)...)
	if i.GPU == "" {
		vms = append(vms,
			&api.ContainerVolumeMount{
				UniqueName: "nvidia-driver-vol",
				Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
				MountPath:  "/usr/nvidia",
				HostPath: &api.ContainerVolumeMountHostPath{
					Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
					Path: "/var/lib/docker/volumes/nvidia-driver-vol/_data",
				},
			})
	}

	return &compute.PodContainerCreateInput{
		ContainerSpec: compute.ContainerSpec{
			ContainerSpec: api.ContainerSpec{
				EnableLxcfs:     i.EnableLxcfs,
				Image:           i.WolfImage,
				ImagePullPolicy: api.ImagePullPolicyAlways,
				CgroupDevicesAllow: []string{
					CGROUP_RULE_13,
				},
				Envs: envs,
			},
			Devices:      devs,
			VolumeMounts: vms,
		},
	}
}

func NewAppSteamContainer(i CreateInput) *compute.PodContainerCreateInput {
	// TODO: 设置 ulimit 和 ipc host
	// --ipc host --ulimit nofile=10240:10240
	zero := 0

	devs := []*compute.ContainerDevice{
		NewHostDev(DEV_UINPUT),
		NewHostDev(DEV_UHID),
	}

	envs := []*api.ContainerKeyValue{
		NewEnv("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"),
		NewEnv("UNAME", "retro"),
		NewEnv("UMASK", "000"),
		NewEnv("HOME", "/home/retro"),
		NewEnv("TZ", "Europe/London"),
		NewEnv("NEEDRESTART_SUSPEND", "1"),
		NewEnv("GAMESCOPE_VERSION", "3.15.14"),
		NewEnv("BUILD_ARCHITECTURE", "amd64"),
		NewEnv("DEBIAN_FRONTEND", "noninteractive"),
		NewEnv("DEB_BUILD_OPTIONS", "noddeb"),
		NewEnv("XDG_RUNTIME_DIR", "/tmp/sockets"),
	}
	if steamNoBigScreen {
		envs = append(envs, NewEnv("STEAM_STARTUP_FLAGS", "-fullscreen"))
	}

	if i.GPU == "" && i.GPUEnvId == "" && i.GPUModel == "" && i.GPUType == "" {
		devs = append(devs, getNvidiaAppDevs(0)...)
	} else if i.GPUEnvId != "" {
		envs = append(envs,
			NewEnv("NVIDIA_VISIBLE_DEVICES", i.GPUEnvId),
			NewEnv("NVIDIA_DRIVER_CAPABILITIES", "all"))
	} else {
		devs = append(devs, &compute.ContainerDevice{
			Type: api.CONTAINER_DEVICE_TYPE_ISOLATED_DEVICE,
			IsolatedDevice: &compute.ContainerIsolatedDevice{
				Index: &zero,
			},
		})
	}

	dataVol := &api.ContainerVolumeMount{
		UniqueName: "home-data",
		Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_DISK,
		MountPath:  "/home/retro",
		Disk: &api.ContainerVolumeMountDisk{
			Index:        &zero,
			SubDirectory: "home",
		},
	}
	if i.Overlay != "" {
		overlayParts := strings.Split(i.Overlay, ":")
		dataVol.Disk.Overlay = &api.ContainerVolumeMountDiskOverlay{
			LowerDir: overlayParts,
		}
	}

	vols := []*api.ContainerVolumeMount{
		{
			UniqueName: "fake-udev",
			Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
			MountPath:  "/usr/bin/fake-udev",
			HostPath: &api.ContainerVolumeMountHostPath{
				Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE,
				Path: "/etc/wolf/fake-udev",
			},
			ReadOnly: true,
		},
		dataVol,
		/*{
			UniqueName: "home-data",
			Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
			MountPath:  "/home/retro",
			HostPath: &api.ContainerVolumeMountHostPath{
				Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
				Path: "/etc/wolf/18046928878093460462/Steam",
			},
		},*/
		{
			UniqueName: "steam-tmp-sockets",
			Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
			MountPath:  "/tmp/sockets",
			HostPath: &api.ContainerVolumeMountHostPath{
				Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
				Path: getTmpSocketsHostPath(i.Name),
			},
		},
		// {
		// 	UniqueName: "steam-tmp-pulse-audio",
		// 	Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
		// 	MountPath:  "/tmp/sockets/pulse-socket",
		// 	HostPath: &api.ContainerVolumeMountHostPath{
		// 		Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE,
		// 		Path: "/tmp/sockets/pulse-socket",
		// 	},
		// 	// Propagation: api.MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL,
		// },
		{
			UniqueName: "steam-run-udev",
			Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
			MountPath:  VOL_RUN_UDEV,
			HostPath: &api.ContainerVolumeMountHostPath{
				Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
				Path: "/etc/wolf/18046928878093460462/Steam/udev",
			},
		},
	}
	vols = append(vols, getMounts(i.MountList)...)
	vols = append(vols, getMounts(i.AppMountList)...)
	if i.GPU == "" || i.AlwaysMountDriverVol {
		vols = append(vols,
			&api.ContainerVolumeMount{
				UniqueName: "steam-nvidia-driver-vol",
				Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
				MountPath:  "/usr/nvidia",
				HostPath: &api.ContainerVolumeMountHostPath{
					Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
					Path: "/var/lib/docker/volumes/nvidia-driver-vol/_data",
				},
			},
		)
	}

	return &compute.PodContainerCreateInput{
		ContainerSpec: compute.ContainerSpec{
			ContainerSpec: api.ContainerSpec{
				AlwaysRestart:   true,
				EnableLxcfs:     i.EnableLxcfs,
				Image:           i.SteamImage,
				ImagePullPolicy: api.ImagePullPolicyAlways,
				Command: []string{"/opt/bin/wolf-hook",
					"-addr", "127.0.0.1",
					"-ulimit-nofile-hard", "524288",
					"-ulimit-nofile-soft", "524288",
				},
				CgroupDevicesAllow: []string{
					CGROUP_RULE_13,
					CGROUP_RULE_244,
				},
				Envs: envs,
				Capabilities: &api.ContainerCapability{
					Add: []string{"SYS_ADMIN", "SYS_NICE", "SYS_PTRACE", "NET_RAW", "MKNOD", "NET_ADMIN"},
				},
			},
			Devices:      devs,
			VolumeMounts: vols,
		},
	}
}

func HTTPSPort(basePort int) int {
	return basePort
}

func HTTPPort(basePort int) int {
	return basePort + 5
}

func ControlUDPPort(basePort int) int {
	return basePort + 15
}

func VideoUDPPingPort(basePort int) int {
	return basePort + 116
}

func AudioUDPPingPort(basePort int) int {
	return basePort + 216
}

func RTSPTCPSetupPort(basePort int) int {
	return basePort + 26
}

func getPortEnvs(basePort int) []*api.ContainerKeyValue {
	httpsPort := HTTPSPort(basePort)
	httpPort := HTTPPort(basePort)
	controlUDPPort := ControlUDPPort(basePort)
	videoUDPPingPort := VideoUDPPingPort(basePort)
	audioUDPPingPort := AudioUDPPingPort(basePort)
	rtspTCPSetupPort := RTSPTCPSetupPort(basePort)

	newV := func(key string, port int) *api.ContainerKeyValue {
		return &api.ContainerKeyValue{
			Key:   key,
			Value: fmt.Sprintf("%d", port),
		}
	}

	return []*api.ContainerKeyValue{
		newV("WOLF_HTTP_PORT", httpPort),
		newV("WOLF_HTTPS_PORT", httpsPort),
		newV("WOLF_CONTROL_PORT", controlUDPPort),
		newV("WOLF_RTSP_SETUP_PORT", rtspTCPSetupPort),
		newV("WOLF_VIDEO_PING_PORT", videoUDPPingPort),
		newV("WOLF_AUDIO_PING_PORT", audioUDPPingPort),
	}
}

func getPortMappings(basePort int) compute.GuestPortMappings {
	httpsPort := HTTPSPort(basePort)
	httpPort := HTTPPort(basePort)
	controlUDPPort := ControlUDPPort(basePort)
	videoUDPPingPort := VideoUDPPingPort(basePort)
	audioUDPPingPort := AudioUDPPingPort(basePort)
	rtspTCPSetupPort := RTSPTCPSetupPort(basePort)
	return compute.GuestPortMappings{
		{
			// HTTPS
			Protocol: "tcp",
			Port:     httpsPort,
			HostPort: &httpsPort,
		},
		{
			// HTTP
			Protocol: "tcp",
			Port:     httpPort,
			HostPort: &httpPort,
		},
		{
			// Control UDP
			Protocol: "udp",
			Port:     controlUDPPort,
			HostPort: &controlUDPPort,
		},
		{
			// Video UDP Ping
			Protocol: "udp",
			Port:     videoUDPPingPort,
			HostPort: &videoUDPPingPort,
		},
		{
			// Audio UDP Ping
			Protocol: "udp",
			Port:     audioUDPPingPort,
			HostPort: &audioUDPPingPort,
		},
		{
			// RTSP TCP Setup
			Protocol: "tcp",
			Port:     rtspTCPSetupPort,
			HostPort: &rtspTCPSetupPort,
		},
	}
}

type CreateInput struct {
	Name         string
	BasePort     int
	NCPU         int
	MemGB        int
	DiskSizeGB   int
	Network      string
	IP           string
	GPU          string
	GPUType      string
	GPUModel     string
	GPUEnvId     string
	EnableLxcfs  bool
	WolfImage    string
	WolfAllGpu   bool
	MountList    []string
	AppMountList []string
	// RenderNode           string
	ExternalIP           string
	Overlay              string
	AlwaysMountDriverVol bool
	SteamImage           string
}

func GetCreateParams(i CreateInput) *compute.ServerCreateInput {
	input := &compute.ServerCreateInput{
		ServerConfigs: &compute.ServerConfigs{
			Hypervisor: compute.HYPERVISOR_POD,
		},
	}
	input.Name = i.Name
	input.VcpuCount = i.NCPU
	input.VmemSize = i.MemGB * 1024
	fv := false
	input.DisableDelete = &fv
	input.AutoStart = true
	input.Disks = []*compute.DiskConfig{
		{
			SizeMb: i.DiskSizeGB * 1024,
			Format: "raw",
			Fs:     "ext4",
		},
	}
	net := &compute.NetworkConfig{
		Network:      i.Network,
		Address:      i.IP,
		PortMappings: getPortMappings(i.BasePort),
	}
	input.Networks = []*compute.NetworkConfig{net}
	input.Pod = &compute.PodCreateInput{
		HostIPC: true,
		Containers: []*compute.PodContainerCreateInput{
			NewPulseAudioContainer(i.Name, i.EnableLxcfs),
			NewWolfContainer(i),
			NewAppSteamContainer(i),
		},
	}
	if i.GPU != "" || i.GPUModel != "" || i.GPUType != "" {
		input.IsolatedDevices = []*compute.IsolatedDeviceConfig{
			{
				Id:      i.GPU,
				DevType: i.GPUType,
				Model:   i.GPUModel,
			},
		}
	}
	return input
}

func getSession() *mcclient.ClientSession {
	client := mcclient.NewClient(authUrl, 60, true, true, "", "")
	token, err := client.Authenticate(user, password, "Default", "system", "Default")
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}
	s := client.NewSession(context.Background(), region, "", "publicURL", token)
	return s
}

func main() {
	s := getSession()
	input := CreateInput{
		Name:         podName,
		BasePort:     basePort,
		NCPU:         ncpu,
		MemGB:        mem,
		DiskSizeGB:   diskSizeGB,
		Network:      podNet,
		IP:           podIP,
		GPU:          gpu,
		GPUEnvId:     gpuEnvId,
		GPUModel:     gpuModel,
		GPUType:      gpuType,
		EnableLxcfs:  enableLxcfs,
		WolfImage:    wolfImage,
		WolfAllGpu:   wolfAllGpu,
		MountList:    mountList,
		AppMountList: appMountList,
		// RenderNode:           renderNode,
		ExternalIP:           externalIP,
		Overlay:              overlay,
		AlwaysMountDriverVol: alwaysMountDriverVol,
		SteamImage:           steamImage,
	}
	obj, err := modules.Servers.Create(s, jsonutils.Marshal(GetCreateParams(input)))
	if err != nil {
		log.Errorf("Failed to create server: %v", err)
		return
	}
	log.Infof("Created server: %v", obj.PrettyString())
	hostId := ""
	var srvDetails *compute.ServerDetails
	for hostId == "" {
		time.Sleep(1 * time.Second)
		id, _ := obj.GetString("id")
		log.Infof("Waiting for server to be scheduled to host...")
		obj, err = modules.Servers.Get(s, id, nil)
		if err != nil {
			log.Errorf("Failed to get server: %v", err)
			return
		}
		guestDetails := &compute.ServerDetails{}
		if err := obj.Unmarshal(guestDetails); err != nil {
			log.Errorf("Failed to unmarshal guest details: %v", err)
			return
		}
		for _, failMsg := range []string{"failed", "fail"} {
			if strings.Contains(guestDetails.Status, failMsg) {
				log.Errorf("Server creation failed: %s", guestDetails.Status)
				return
			}
		}
		hostId = guestDetails.HostId
		srvDetails = guestDetails
	}
	accessIp := srvDetails.HostEIP
	if accessIp == "" {
		accessIp = srvDetails.HostAccessIp
	}
	if accessIp == "" {
		accessIp = srvDetails.IPs
	}
	log.Infof("Access URL: %s:%d , port_mappings: %s", accessIp, accessPort, getPortMappings(input.BasePort).String())
}
