package kubelet

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContainerdRuntimeInfo(t *testing.T) {
	info := &RuntimeInfo{RootDir: containerdRootDir}
	assert.Equal(t, "/var/lib/containerd", info.RootDir)
}

func TestParseDockerRuntimeInfo(t *testing.T) {
	info, err := parseDockerRuntimeInfo([]byte(`{"DockerRootDir":"/var/lib/docker"}`))
	if assert.NoError(t, err) {
		assert.Equal(t, "/var/lib/docker", info.RootDir)
	}

	_, err = parseDockerRuntimeInfo([]byte(`{"DockerRootDir":""}`))
	assert.Error(t, err)
}
