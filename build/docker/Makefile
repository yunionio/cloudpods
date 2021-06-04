REGISTRY ?= "registry.cn-beijing.aliyuncs.com/yunionio"

DOCKER_BUILD = docker build -t $(REGISTRY)
DOCKER_BUILDX = docker buildx build --platform linux/arm64,linux/amd64 --push -t $(REGISTRY)

debian10-base:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/debian10-base:1.0 -f ./Dockerfile.debian-base .

climc-base:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/climc-base:$(VERSION) -f ./Dockerfile.climc-base .

BAREMETAL_BASE_VERSION = v0.3

baremetal-base:
	$(DOCKER_BUILD)/baremetal-base:$(BAREMETAL_BASE_VERSION) -f ./Dockerfile.baremetal-base .
	docker push $(REGISTRY)/baremetal-base:$(BAREMETAL_BASE_VERSION)
