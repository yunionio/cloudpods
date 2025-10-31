REGISTRY ?= "registry.cn-beijing.aliyuncs.com/yunionio"
DOCKER_BUILD = docker build -t $(REGISTRY)
DOCKER_BUILDX = docker buildx build --platform linux/arm64,linux/amd64,linux/riscv64 --push -t $(REGISTRY)


debian10-base:
	docker buildx build --platform linux/arm64,linux/amd64 --push \
		-t registry.cn-beijing.aliyuncs.com/yunionio/debian10-base:1.0 -f ./Dockerfile.debian-base .

ONECLOUD_BASE_VERSION = v0.3-3.13.5
ONECLOUD_BASE_VERSION_3-15-4 = v3.15.4-0
ONECLOUD_BASE_VERSION_3-22-2 = 3.22.2-0

onecloud-base:
	$(DOCKER_BUILDX)/onecloud-base:$(ONECLOUD_BASE_VERSION) -f ./Dockerfile.onecloud-base .
onecloud-base-3-15-4:
	$(DOCKER_BUILDX)/onecloud-base:$(ONECLOUD_BASE_VERSION_3-15-4) -f ./Dockerfile.onecloud-base-3-15-4 .
onecloud-base-3-22-2:
	$(DOCKER_BUILDX)/onecloud-base:$(ONECLOUD_BASE_VERSION_3-22-2) -f ./Dockerfile.onecloud-base-3-22-2 .

ANSIBLESERVER_BASE = 3.22.2-0

ansibleserver-base:
	$(DOCKER_BUILDX)/ansibleserver-base:$(ANSIBLESERVER_BASE) -f ./Dockerfile.ansibleserver-base .

CLIMC_BASE_VERSION = 20230731.5
CLIMC_BASE_VERSION_3-22-2 = 3.22.2-2

climc-base:
	$(DOCKER_BUILDX)/climc-base:$(CLIMC_BASE_VERSION) -f ./Dockerfile.climc-base .

climc-base-3-22-2:
	$(DOCKER_BUILDX)/climc-base:$(CLIMC_BASE_VERSION_3-22-2) -f ./Dockerfile.climc-base-3-22-2 .

KUBECTL_VERSION_3-22-2 = 3.22.2-1
kubectl:
	$(DOCKER_BUILDX)/kubectl:$(KUBECTL_VERSION_3-22-2) -f ./Dockerfile.kubectl .

WEBCONSOLE_BASE_VERSION_3-22-2 = 3.22.2-1
webconsole-base:
	$(DOCKER_BUILDX)/webconsole-base:$(WEBCONSOLE_BASE_VERSION_3-22-2) -f ./Dockerfile.webconsole-base .

BAREMETAL_BASE_VERSION = v0.3.9-20251112.1

FEDORA_RISCV64_VERSION = 42
fedora-riscv64-base:
	wget -q https://dl.fedoraproject.org/pub/alt/risc-v/release/$(FEDORA_RISCV64_VERSION)/Container/riscv64/images/Fedora-Container-Base-Generic-$(FEDORA_RISCV64_VERSION).20250414-8635a3a5bfcd.riscv64.oci.tar.xz && \
	xz -d Fedora-Container-Base-Generic-$(FEDORA_RISCV64_VERSION).20250414-8635a3a5bfcd.riscv64.oci.tar.xz && \
	skopeo copy oci-archive:Fedora-Container-Base-Generic-$(FEDORA_RISCV64_VERSION).20250414-8635a3a5bfcd.riscv64.oci.tar docker-daemon:fedora-riscv64:$(FEDORA_RISCV64_VERSION) && \
	rm -f Fedora-Container-Base-Generic-$(FEDORA_RISCV64_VERSION).20250414-8635a3a5bfcd.riscv64.oci.tar

baremetal-base-riscv: fedora-riscv64-base
         $(DOCKER_BUILDX)/baremetal-base:$(BAREMETAL_BASE_VERSION) -f ./Dockerfile.baremetal-base-riscv .
         #docker push $(REGISTRY)/baremetal-base:$(BAREMETAL_BASE_VERSION)

baremetal-base:
	$(DOCKER_BUILDX)/baremetal-base:$(BAREMETAL_BASE_VERSION) -f ./Dockerfile.baremetal-base .
	#docker push $(REGISTRY)/baremetal-base:$(BAREMETAL_BASE_VERSION)

TORRENT_VERSION = 20210815.0
TORRENT_VERSION_3-22-2 = 3.22.2-0
torrent:
	$(DOCKER_BUILDX)/torrent:$(TORRENT_VERSION) -f ./Dockerfile.torrent .

GLANCE_BASE_VERSION = v0.0.1
GLANCE_BASE_VERSION_3-22-2 = 3.22.2-0
glance-base:
	$(DOCKER_BUILDX)/glance-base:$(GLANCE_BASE_VERSION) -f ./Dockerfile.glance-base .

glance-base-3-22-2:
	$(DOCKER_BUILDX)/glance-base:$(GLANCE_BASE_VERSION_3-22-2) -f ./Dockerfile.glance-base .


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

LBAGENT_BASE_VERSION = v0.0.5
lbagent-base:
	$(DOCKER_BUILDX)/lbagent-base:$(LBAGENT_BASE_VERSION) -f ./Dockerfile.lbagent-base .

GUACD_VERSION=1.6.0
guacd:
	$(DOCKER_BUILDX)/guacd:$(GUACD_VERSION) -f ./Dockerfile.guacd .

