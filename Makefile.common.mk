ifeq ($(__inc_Makefile_common_mk),)
__inc_Makefile_common_mk:=1

ifeq ($(ModName),)
  $(error ModName must be set, e.g. yunion.io/x/onecloud)
endif

ModBaseName:=$(notdir $(ModName))

DockerImageRegistry?=registry.cn-beijing.aliyuncs.com
DockerImageAlpineBuild?=$(DockerImageRegistry)/yunionio/alpine-build:3.19.0-go-1.21.10-0
DockerImageCentOSBuild?=$(DockerImageRegistry)/yunionio/centos-build:go-1.21.10-0

EnvIf=$(if $($(1)),$(1)=$($(1)))

define dockerCentOSBuildCmd
set -o xtrace
set -o errexit
set -o pipefail
git config --global --add safe.directory /root/go/src/yunion.io/x/$(ModBaseName)
cd /root/go/src/yunion.io/x/$(ModBaseName)
env \
	$(call EnvIf,GOARCH) \
	$(call EnvIf,GOOS) \
	$(call EnvIf,CGO_ENABLED) \
	make $(1)
chown -R $(shell id -u):$(shell id -g) _output
endef

tmpName=$(ModBaseName)-$(shell date +"%Y%m%d.%H%M%S%3N")

docker-centos-build: export dockerCentOSBuildCmd:=$(call dockerCentOSBuildCmd,$(F))
docker-centos-build:
	docker rm --force docker-centos-build-$(tmpName) &>/dev/null || true
	docker run \
		--rm \
		--name docker-centos-build-$(tmpName) \
		-v $(CURDIR):/root/go/src/yunion.io/x/$(ModBaseName) \
		-v $(CURDIR)/_output/centos-build:/root/go/src/yunion.io/x/$(ModBaseName)/_output \
		-v $(CURDIR)/_output/centos-build/_cache:/root/.cache \
		$(DockerImageCentOSBuild) \
		/bin/bash -c "$$dockerCentOSBuildCmd"
	ls -lh _output/centos-build/bin

# NOTE we need a way to stop and remove the container started by docker-build.
# No --tty, --stop-signal won't work
docker-centos-build-stop:
	docker stop --time 0 docker-centos-build-$(tmpName) || true

.PHONY: docker-centos-build
.PHONY: docker-centos-build-stop


define dockerAlpineBuildCmd
set -o xtrace
set -o errexit
set -o pipefail
git config --global --add safe.directory /root/go/src/yunion.io/x/$(ModBaseName)
cd /root/go/src/yunion.io/x/$(ModBaseName)
git config --global --add safe.directory /root/go/src/yunion.io/x/onecloud
env \
	$(call EnvIf,GOARCH) \
	$(call EnvIf,GOOS) \
	$(call EnvIf,CGO_ENABLED) \
	make $(1)
chown -R $(shell id -u):$(shell id -g) _output
endef

docker-alpine-build: export dockerAlpineBuildCmd:=$(call dockerAlpineBuildCmd,$(F))
docker-alpine-build:
	docker rm --force docker-alpine-build-$(tmpName) &>/dev/null || true
	docker run \
		--rm \
		--name docker-alpine-build-$(tmpName) \
		-v $(CURDIR):/root/go/src/yunion.io/x/$(ModBaseName) \
		-v $(CURDIR)/_output/alpine-build:/root/go/src/yunion.io/x/$(ModBaseName)/_output \
		-v $(CURDIR)/_output/alpine-build/_cache:/root/.cache \
		$(DockerImageAlpineBuild) \
		/bin/sh -c "$$dockerAlpineBuildCmd"
	ls -lh _output/alpine-build/bin

docker-alpine-build-stop:
	docker stop --time 0 docker-alpine-build-$(tmpName) || true

.PHONY: docker-alpine-build
.PHONY: docker-alpine-build-stop

endif # __inc_Makefile_common_mk
