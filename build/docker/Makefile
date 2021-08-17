REGISTRY ?= "registry.cn-beijing.aliyuncs.com/yunionio"
DOCKER_BUILD = docker build -t $(REGISTRY)
DOCKER_BUILDX = docker buildx build --platform linux/arm64,linux/amd64 --push -t $(REGISTRY)


debian10-base:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/debian10-base:1.0 -f ./Dockerfile.debian-base .

ONECLOUD_BASE_VERSION = v0.3.5

onecloud-base:
	$(DOCKER_BUILDX)/onecloud-base:$(ONECLOUD_BASE_VERSION) -f ./Dockerfile.onecloud-base .

ANSIBLESERVER_BASE = v1.0.5

ansibleserver-base:
	$(DOCKER_BUILDX)/ansibleserver-base:$(ANSIBLESERVER_BASE) -f ./Dockerfile.ansibleserver-base .

CLIMC_BASE_VERSION = 20210817

climc-base:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/climc-base:$(CLIMC_BASE_VERSION) -f ./Dockerfile.climc-base .

BAREMETAL_BASE_VERSION = v0.3.2

baremetal-base:
	$(DOCKER_BUILDX)/baremetal-base:$(BAREMETAL_BASE_VERSION) -f ./Dockerfile.baremetal-base .
	#docker push $(REGISTRY)/baremetal-base:$(BAREMETAL_BASE_VERSION)

FILE_REPO_VERSION = v0.3.3

file-repo:
	$(DOCKER_BUILDX)/file-repo:$(FILE_REPO_VERSION) -f ./Dockerfile.file-repo .
