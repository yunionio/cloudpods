package main

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

const (
	NV_DEV_UVM       = "/dev/nvidia-uvm"
	NV_DEV_UVM_TOOLS = "/dev/nvidia-uvm-tools"

	DEV_DRI            = "/dev/dri/"
	DEV_DRI_CARD0      = "/dev/dri/card0"
	DEV_DRI_RENDERD128 = "/dev/dri/renderD128"

	NV_DEV_CAP1    = "/dev/nvidia-caps/nvidia-cap1"
	NV_DEV_CAP2    = "/dev/nvidia-caps/nvidia-cap2"
	NV_DEV_CTL     = "/dev/nvidiactl"
	NV_DEV_NVIDIA0 = "/dev/nvidia0"
	NV_DEV_MODESET = "/dev/nvidia-modeset"
	DEV_UINPUT     = "/dev/uinput"
	DEV_UHID       = "/dev/uhid"

	VOL_DEV      = "/dev"
	VOL_RUN_UDEV = "/run/udev"

	CGROUP_RULE_13  = "c 13:* rmw"
	CGROUP_RULE_244 = "c 244:* rmw"
)

var (
	IP = "192.168.6.70"
)

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

func NewWolfContainer() *compute.PodContainerCreateInput {
	zero := 0
	return &compute.PodContainerCreateInput{
		Name: "wolf",
		ContainerSpec: compute.ContainerSpec{
			ContainerSpec: api.ContainerSpec{
				Image: "registry.cn-beijing.aliyuncs.com/zexi/wolf:hook.11",
				CgroupDevicesAllow: []string{
					CGROUP_RULE_13,
				},
				Envs: []*api.ContainerKeyValue{
					NewEnv("WOLF_LOG_LEVEL", "DEBUG"),
					NewEnv("WOLF_BASE_PORT", "20105"),
					NewEnv("WOLF_EXTERNAL_IP", IP),
					NewEnv("NVIDIA_DRIVER_VOLUME_NAME", "nvidia-driver-vol"), // FIXME: not need?
					NewEnv("HOST_APPS_STATE_FOLDER", "/etc/wolf"),
					NewEnv("XDG_RUNTIME_DIR", "/tmp/sockets"),
				},
			},
			Devices: []*compute.ContainerDevice{
				NewHostDev(NV_DEV_UVM),
				NewHostDev(NV_DEV_UVM_TOOLS),
				NewHostDev(DEV_DRI),
				NewHostDev(NV_DEV_CAP1),
				NewHostDev(NV_DEV_CAP2),
				NewHostDev(NV_DEV_CTL),
				NewHostDev(NV_DEV_NVIDIA0),
				NewHostDev(NV_DEV_MODESET),
				NewHostDev(DEV_UINPUT),
				NewHostDev(DEV_UHID),
			},
			VolumeMounts: []*api.ContainerVolumeMount{
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
						Index: &zero,
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
					Propagation: api.MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL,
				},
				{
					UniqueName: "dev",
					Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
					MountPath:  VOL_DEV,
					HostPath: &api.ContainerVolumeMountHostPath{
						Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
						Path: VOL_DEV,
					},
					Propagation: api.MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL,
				},
				{
					UniqueName: "tmp-sockets",
					Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
					MountPath:  "/tmp/sockets",
					HostPath: &api.ContainerVolumeMountHostPath{
						Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
						Path: "/tmp/sockets",
					},
					Propagation: api.MOUNTPROPAGATION_PROPAGATION_BIDIRECTIONAL,
				},
				{
					UniqueName: "nvidia-driver-vol",
					Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
					MountPath:  "/usr/nvidia",
					HostPath: &api.ContainerVolumeMountHostPath{
						Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
						Path: "/var/lib/docker/volumes/nvidia-driver-vol/_data",
					},
				},
				{
					UniqueName: "docker-socket",
					Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
					MountPath:  "/var/run/docker.sock",
					HostPath: &api.ContainerVolumeMountHostPath{
						Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_FILE,
						Path: "/var/run/docker.sock",
					},
				},
			},
		},
	}
}

func NewAppSteamContainer() *compute.PodContainerCreateInput {
	// TODO: 设置 ulimit 和 ipc host
	// --ipc host --ulimit nofile=10240:10240
	return &compute.PodContainerCreateInput{
		Name: "steam",
		ContainerSpec: compute.ContainerSpec{
			ContainerSpec: api.ContainerSpec{
				Image:           "registry.cn-beijing.aliyuncs.com/zexi/steam:custom.0",
				ImagePullPolicy: api.ImagePullPolicyAlways,
				Command:         []string{"/opt/bin/wolf-hook", "-addr", "0.0.0.0"},
				CgroupDevicesAllow: []string{
					CGROUP_RULE_13,
					CGROUP_RULE_244,
				},
				Envs: []*api.ContainerKeyValue{
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
				},
				Capabilities: &api.ContainerCapability{
					Add: []string{"SYS_ADMIN", "SYS_NICE", "SYS_PTRACE", "NET_RAW", "MKNOD", "NET_ADMIN"},
				},
			},
			Devices: []*compute.ContainerDevice{
				NewHostDev(NV_DEV_UVM),
				NewHostDev(NV_DEV_UVM_TOOLS),
				NewHostDev(DEV_DRI_CARD0),
				NewHostDev(DEV_DRI_RENDERD128),
				NewHostDev(NV_DEV_CAP1),
				NewHostDev(NV_DEV_CAP2),
				NewHostDev(NV_DEV_CTL),
				NewHostDev(NV_DEV_NVIDIA0),
				NewHostDev(NV_DEV_MODESET),
				NewHostDev(DEV_UINPUT),
				NewHostDev(DEV_UHID),
			},
			VolumeMounts: []*api.ContainerVolumeMount{
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
				{
					UniqueName: "steam-nvidia-driver-vol",
					Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
					MountPath:  "/usr/nvidia",
					HostPath: &api.ContainerVolumeMountHostPath{
						Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
						Path: "/var/lib/docker/volumes/nvidia-driver-vol/_data",
					},
				},
				{
					UniqueName: "home-data",
					Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
					MountPath:  "/home/retro",
					HostPath: &api.ContainerVolumeMountHostPath{
						Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
						Path: "/etc/wolf/18046928878093460462/Steam",
					},
				},
				{
					UniqueName: "steam-tmp-sockets",
					Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
					MountPath:  "/tmp/sockets",
					HostPath: &api.ContainerVolumeMountHostPath{
						Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
						Path: "/tmp/sockets",
					},
				},
				{
					UniqueName: "steam-run-udev",
					Type:       api.CONTAINER_VOLUME_MOUNT_TYPE_HOST_PATH,
					MountPath:  VOL_RUN_UDEV,
					HostPath: &api.ContainerVolumeMountHostPath{
						Type: api.CONTAINER_VOLUME_MOUNT_HOST_PATH_TYPE_DIRECTORY,
						Path: "/etc/wolf/18046928878093460462/Steam/udev",
					},
				},
			},
		},
	}
}

func GetCreateParams(name string) *compute.ServerCreateInput {
	input := &compute.ServerCreateInput{
		ServerConfigs: &compute.ServerConfigs{
			Hypervisor: compute.HYPERVISOR_POD,
		},
	}
	input.Name = name
	input.VcpuCount = 8
	input.VmemSize = 10240
	fv := false
	input.DisableDelete = &fv
	input.AutoStart = true
	input.Disks = []*compute.DiskConfig{
		{
			SizeMb: 1024,
			Format: "raw",
			Fs:     "ext4",
		},
	}
	input.Networks = []*compute.NetworkConfig{
		{
			Network: "cnet-6",
			Address: IP,
		},
	}
	input.Pod = &compute.PodCreateInput{
		HostIPC: true,
		Containers: []*compute.PodContainerCreateInput{
			NewWolfContainer(),
			NewAppSteamContainer(),
		},
	}
	return input
}

func getSession() *mcclient.ClientSession {
	client := mcclient.NewClient("https://10.168.26.9:30357/v3", 60, true, true, "", "")
	token, err := client.Authenticate("sysadmin", "X9WGwepHnu7yVxwh", "Default", "system", "Default")
	if err != nil {
		log.Fatalf("Failed to authenticate: %v", err)
	}
	s := client.NewSession(context.Background(), "region0", "", "publicURL", token)
	return s
}

func main() {
	s := getSession()
	obj, err := modules.Servers.Create(s, jsonutils.Marshal(GetCreateParams("lzx-wolf-app")))
	if err != nil {
		log.Errorf("Failed to create server: %v", err)
		return
	}
	log.Infof("Created server: %v", obj.PrettyString())
}
