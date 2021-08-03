REGISTRY ?= "registry.cn-beijing.aliyuncs.com/yunionio"
DOCKER_BUILD = docker build -t $(REGISTRY)
DOCKER_BUILDX = docker buildx build --platform linux/arm64,linux/amd64 --push -t $(REGISTRY)


debian10-base:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/debian10-base:1.0 -f ./Dockerfile.debian-base .

CLIMC_BASE_VERSION = 20210701

climc-base:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/climc-base:$(CLIMC_BASE_VERSION) -f ./Dockerfile.climc-base .

BAREMETAL_BASE_VERSION = v0.3.2

baremetal-base:
	$(DOCKER_BUILDX)/baremetal-base:$(BAREMETAL_BASE_VERSION) -f ./Dockerfile.baremetal-base .
	#docker push $(REGISTRY)/baremetal-base:$(BAREMETAL_BASE_VERSION)
