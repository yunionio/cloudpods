REGISTRY ?= "registry.cn-beijing.aliyuncs.com/yunionio"
DOCKER_BUILD = docker build -t $(REGISTRY)
DOCKER_BUILDX = docker buildx build --platform linux/arm64,linux/amd64 --push -t $(REGISTRY)


debian10-base:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/debian10-base:1.0 -f ./Dockerfile.debian-base .

ONECLOUD_BASE_VERSION = v0.3-3.13.5
ONECLOUD_BASE_VERSION_3-15-4 = v3.15.4-0

onecloud-base:
	$(DOCKER_BUILDX)/onecloud-base:$(ONECLOUD_BASE_VERSION) -f ./Dockerfile.onecloud-base .
onecloud-base-3-15-4:
	$(DOCKER_BUILDX)/onecloud-base:$(ONECLOUD_BASE_VERSION_3-15-4) -f ./Dockerfile.onecloud-base-3-15-4 .

ANSIBLESERVER_BASE = v1.1.2

ansibleserver-base:
	$(DOCKER_BUILDX)/ansibleserver-base:$(ANSIBLESERVER_BASE) -f ./Dockerfile.ansibleserver-base .

CLIMC_BASE_VERSION = 20230731.5

climc-base:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/climc-base:$(CLIMC_BASE_VERSION) -f ./Dockerfile.climc-base .

KUBECTL_VERSION = 1.22.9
kubectl:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/kubectl:$(KUBECTL_VERSION) -f ./Dockerfile.kubectl .

WEBCONSOLE_BASE_VERSION = 20240909.0
webconsole-base:
	$(DOCKER_BUILDX)/webconsole-base:$(WEBCONSOLE_BASE_VERSION) -f ./Dockerfile.webconsole-base .

BAREMETAL_BASE_VERSION = v0.3.9-20231219.1

baremetal-base:
	$(DOCKER_BUILDX)/baremetal-base:$(BAREMETAL_BASE_VERSION) -f ./Dockerfile.baremetal-base .
	#docker push $(REGISTRY)/baremetal-base:$(BAREMETAL_BASE_VERSION)

TORRENT_VERSION = 20210815.0
torrent:
	$(DOCKER_BUILDX)/torrent:$(TORRENT_VERSION) -f ./Dockerfile.torrent .

GLANCE_BASE_VERSION = v0.0.1
glance-base:
	$(DOCKER_BUILDX)/glance-base:$(GLANCE_BASE_VERSION) -f ./Dockerfile.glance-base .

FILE_REPO_VERSION = v0.5.2

file-repo:
	$(DOCKER_BUILDX)/file-repo:$(FILE_REPO_VERSION) -f ./Dockerfile.file-repo .

HOST_IMAGE_VERSION ?= v1.0.2
HOST_IMAGE_NAME = $(REGISTRY)/host-image:$(HOST_IMAGE_VERSION)
host-image:
	docker pull $(HOST_IMAGE_NAME)-amd64 --platform amd64
	docker pull $(HOST_IMAGE_NAME)-arm64 --platform arm64
	docker manifest create $(HOST_IMAGE_NAME) \
		$(HOST_IMAGE_NAME)-amd64 \
		$(HOST_IMAGE_NAME)-arm64
	docker manifest annotate $(HOST_IMAGE_NAME) $(HOST_IMAGE_NAME)-amd64 --arch amd64
	docker manifest annotate $(HOST_IMAGE_NAME) $(HOST_IMAGE_NAME)-arm64 --arch arm64
	docker manifest push $(HOST_IMAGE_NAME)

LBAGENT_BASE_VERSION = v0.0.3
lbagent-base:
	$(DOCKER_BUILDX)/lbagent-base:$(LBAGENT_BASE_VERSION) -f ./Dockerfile.lbagent-base .

GUACD_VERSION=1.5.3
guacd:
	docker buildx build --platform linux/arm64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/guacd:$(GUACD_VERSION)-arm64 -f ./Dockerfile.guacd .
	docker buildx build --platform linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/guacd:$(GUACD_VERSION)-amd64 -f ./Dockerfile.guacd .
	docker buildx imagetools create -t registry.cn-beijing.aliyuncs.com/yunionio/guacd:$(GUACD_VERSION) \
		registry.cn-beijing.aliyuncs.com/yunionio/guacd:$(GUACD_VERSION)-amd64 \
		registry.cn-beijing.aliyuncs.com/yunionio/guacd:$(GUACD_VERSION)-arm64
