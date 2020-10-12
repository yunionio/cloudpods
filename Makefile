#####################################################

REPO_PREFIX := yunion.io/x/onecloud
VERSION_PKG := yunion.io/x/pkg/util/version
ROOT_DIR := $(CURDIR)
BUILD_DIR := $(ROOT_DIR)/_output
BIN_DIR := $(BUILD_DIR)/bin
BUILD_SCRIPT := $(ROOT_DIR)/build/build.sh

ifeq ($(ONECLOUD_CI_BUILD),)
	GIT_COMMIT := $(shell git rev-parse --short HEAD)
	GIT_BRANCH := $(shell git name-rev --name-only HEAD)
	GIT_VERSION := $(shell git describe --always --tags --abbrev=14 $(GIT_COMMIT)^{commit})
	GIT_TREE_STATE := $(shell s=`git status --porcelain 2>/dev/null`; if [ -z "$$s" ]; then echo "clean"; else echo "dirty"; fi)
	BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
else
	GIT_COMMIT:=x
	GIT_BRANCH:=x
	GIT_VERSION:=x
	BUILD_TREE_STATE:=clean
	BUILD_DATE=2099-07-01T07:11:09Z
endif

LDFLAGS := "-w \
	-X $(VERSION_PKG).gitVersion=$(GIT_VERSION) \
	-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) \
	-X $(VERSION_PKG).gitBranch=$(GIT_BRANCH) \
	-X $(VERSION_PKG).buildDate=$(BUILD_DATE) \
	-X $(VERSION_PKG).gitTreeState=$(GIT_TREE_STATE) \
	-X $(VERSION_PKG).gitMajor=0 \
	-X $(VERSION_PKG).gitMinor=0"


#####################################################
ifneq ($(DLV),)
	GO_BUILD_FLAGS += -gcflags "all=-N -l"
	LDFLAGS = ""
endif
GO_BUILD_FLAGS+=-mod vendor -ldflags $(LDFLAGS)
GO_BUILD := go build $(GO_BUILD_FLAGS)
GO_INSTALL := go install -ldflags $(LDFLAGS)
GO_TEST := go test

PKGS := go list ./...

CGO_CFLAGS_ENV = $(shell go env CGO_CFLAGS)
CGO_LDFLAGS_ENV = $(shell go env CGO_LDFLAGS)

ifdef LIBQEMUIO_PATH
		X_CGO_CFLAGS := ${CGO_CFLAGS_ENV} -I${LIBQEMUIO_PATH}/src -I${LIBQEMUIO_PATH}/src/include
		X_CGO_LDFLAGS := ${CGO_LDFLAGS_ENV} -laio -lqemuio -lpthread  -L ${LIBQEMUIO_PATH}/src
endif

export GOOS ?= linux
export GO111MODULE:=on
export CGO_CFLAGS = ${X_CGO_CFLAGS}
export CGO_LDFLAGS = ${X_CGO_LDFLAGS}

UNAME := $(shell uname)

ifeq ($(UNAME), Linux)
XARGS_FLAGS = --no-run-if-empty
endif

cmdTargets:=$(filter-out cmd/host-image,$(wildcard cmd/*))
rpmTargets:=$(foreach b,$(patsubst cmd/%,%,$(cmdTargets)),$(if $(shell [ -f "$(CURDIR)/build/$(b)/vars" ] && echo 1),rpm/$(b)))

all: build


install: prepare_dir
	@for PKG in $$( $(PKGS) | grep -w "$(filter-out $@,$(MAKECMDGOALS))" ); do \
		echo $$PKG; \
		$(GO_INSTALL) $$PKG; \
	done


gencopyright:
	@bash scripts/gencopyright.sh pkg cmd

test:
	@go test $(GO_BUILD_FLAGS) $(shell go list ./... | egrep -v 'host-image|hostimage')

vet:
	go vet ./...

cmd/esxi-agent: prepare_dir
	CGO_ENABLED=0 $(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@

cmd/%: prepare_dir
	$(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@

rpm/%: cmd/%
	$(BUILD_SCRIPT) $*

pkg/%: prepare_dir
	$(GO_INSTALL) $(REPO_PREFIX)/$@

build:
	$(MAKE) $(cmdTargets)

rpm:
	$(MAKE) $(rpmTargets)

rpmclean:
	rm -fr $(BUILD_DIR)/rpms

prepare_dir: bin_dir


bin_dir: output_dir
	@mkdir -p $(BUILD_DIR)/bin


output_dir:
	@mkdir -p $(BUILD_DIR)


.PHONY: all build prepare_dir clean rpm


clean:
	@rm -fr $(BUILD_DIR)


fmt:
	@git ls-files --exclude '*' '*.go' \
		| grep -v '^vendor/' \
		| xargs $(XARGS_FLAGS) gofmt -w

fmt-check: fmt
	@if git status --short | grep -E '^.M .*/[^.]+.go'; then \
		git diff | cat; \
		echo "$@: working tree modified (possibly by gofmt)" >&2 ; \
		false ; \
	fi
.PHONY: fmt fmt-check

gendocgo:
	@sh build/gendoc.sh

adddocgo:
	@git ls-files --others '*/doc.go' | xargs $(XARGS_FLAGS) -- git add

gendocgo-check: gendocgo
	@n="$$(git ls-files --others '*/doc.go' | wc -l)"; \
	if test "$$n" -gt 0; then \
		git ls-files --others '*/doc.go' | sed -e 's/^/  /'; \
		echo "$@: untracked doc.go file(s) exist in working directory" >&2 ; \
		false ; \
	fi
.PHONY: gendocgo adddocgo gendocgo-check

goimports-check:
	@goimports -w -local "yunion.io/x/:yunion.io/x/onecloud" pkg cmd; \
	if git status --short | grep -E '^.M .*/[^.]+.go'; then \
		git diff | cat; \
		echo "$@: working tree modified (possibly by goimports)" >&2 ; \
		echo "$@: " >&2 ; \
		echo "$@: import spec should be grouped in order: std, 3rd-party, yunion.io/x, yunion.io/x/onecloud" >&2 ; \
		echo "$@: see \"yun\" branch at https://github.com/yousong/tools" >&2 ; \
		false ; \
	fi
.PHONY: goimports-check

vet-check:
	./scripts/vet.sh gen
	./scripts/vet.sh chk
.PHONY: vet-check

check: fmt-check
check: gendocgo-check
check: goimports-check
check: vet-check
.PHONY: check


define depDeprecated
OneCloud now requires using go-mod for dependency management.  dep target,
vendor files will be removed in future versions

Follow the following link to find out more about go-mod

 - https://blog.golang.org/using-go-modules
 - https://github.com/golang/go/wiki/Modules

Switching to "make mod"...

endef

dep: export depDeprecated:=$(depDeprecated)
dep:
	@echo "$$depDeprecated"
	@$(MAKE) mod

mod:
	go get -d $(patsubst %,%@master,$(shell GO111MODULE=on go mod edit -print  | sed -n -e 's|.*\(yunion.io/x/[a-z].*\) v.*|\1|p'))
	go mod tidy
	go mod vendor -v


DOCKER_CENTOS_BUILD_IMAGE?=registry.cn-beijing.aliyuncs.com/yunionio/centos-build:1.1-1

define dockerCentOSBuildCmd
set -o xtrace
set -o errexit
set -o pipefail
cd /root/onecloud
export GOFLAGS=-mod=vendor
make $(1)
chown -R $(shell id -u):$(shell id -g) _output
endef

docker-centos-build: export dockerCentOSBuildCmd:=$(call dockerCentOSBuildCmd,$(F))
docker-centos-build:
	docker rm --force onecloud-ci-build &>/dev/null || true
	docker run \
		--name onecloud-docker-centos-build \
		--rm \
		--volume $(CURDIR):/root/onecloud \
		--volume $(CURDIR)/_output/_cache:/root/.cache \
		$(DOCKER_CENTOS_BUILD_IMAGE) \
		/bin/bash -c "$$dockerCentOSBuildCmd"
	chown -R $$(id -u):$$(id -g) _output
	ls -lh _output/bin

# NOTE we need a way to stop and remove the container started by docker-build.
# No --tty, --stop-signal won't work
docker-centos-build-stop:
	docker stop --time 0 onecloud-docker-centos-build || true

.PHONY: docker-centos-build
.PHONY: docker-centos-build-stop

DOCKER_ALPINE_BUILD_IMAGE?=registry.cn-beijing.aliyuncs.com/yunionio/alpine-build:1.0-3

define dockerAlpineBuildCmd
set -o xtrace
set -o errexit
set -o pipefail
cd /root/go/src/yunion.io/x/onecloud
export GOFLAGS=-mod=vendor
make $(1)
chown -R $(shell id -u):$(shell id -g) _output
endef

docker-alpine-build: export dockerAlpineBuildCmd:=$(call dockerAlpineBuildCmd,$(F))
docker-alpine-build:
	docker rm --force onecloud-docker-alpine-build &>/dev/null || true
	docker run --rm \
		--name onecloud-docker-alpine-build \
		-v $(CURDIR):/root/go/src/yunion.io/x/onecloud \
		-v $(CURDIR)/_output/alpine-build:/root/go/src/yunion.io/x/onecloud/_output \
		-v $(CURDIR)/_output/alpine-build/_cache:/root/.cache \
		$(DOCKER_ALPINE_BUILD_IMAGE) \
		/bin/sh -c "$$dockerAlpineBuildCmd"
	ls -lh _output/alpine-build/bin

docker-alpine-build-stop:
	docker stop --time 0 onecloud-docker-alpine-build || true

.PHONY: docker-alpine-build
.PHONY: docker-alpine-build-stop

define helpText
Build with docker

	make docker-centos-build F='-j4'
	make docker-centos-build F='-j4 cmd/region cmd/climc'
	make docker-centos-build-stop

	make docker-alpine-build F='-j4'
	make docker-alpine-build F='-j4 cmd/host cmd/host-deployer'
	make docker-alpine-build-stop

Tidy up go modules and vendor directory

	make mod
endef

help: export helpText:=$(helpText)
help:
	@echo "$$helpText"

.PHONY: help

gen-model-api-check:
	which model-api-gen || (GO111MODULE=off go get -u yunion.io/x/code-generator/cmd/model-api-gen)

gen-model-api: gen-model-api-check
	$(ROOT_DIR)/scripts/codegen.py model-api

gen-swagger-check:
	which swagger || (GO111MODULE=off go get -u github.com/go-swagger/go-swagger/cmd/swagger)
	which swagger-gen || (GO111MODULE=off go get -u yunion.io/x/code-generator/cmd/swagger-gen)
	which swagger-serve || (GO111MODULE=off go get -u yunion.io/x/code-generator/cmd/swagger-serve)

gen-swagger: gen-swagger-check
	$(ROOT_DIR)/scripts/codegen.py swagger-code
	$(ROOT_DIR)/scripts/codegen.py swagger-yaml

swagger-serve-only:
	$(ROOT_DIR)/scripts/codegen.py swagger-serve

swagger-serve: gen-model-api gen-swagger swagger-serve-only

swagger-site: gen-model-api gen-swagger
	$(ROOT_DIR)/scripts/codegen.py swagger-site

.PHONY: gen-model-api-check gen-model-api gen-swagger-check gen-swagger swagger-serve swagger-site

REGISTRY ?= "registry.cn-beijing.aliyuncs.com/yunionio"
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
                git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)

image: clean
	mkdir -p $(ROOT_DIR)/_output
	DEBUG=$(DEBUG) ARCH=$(ARCH) TAG=$(VERSION) REGISTRY=$(REGISTRY) $(ROOT_DIR)/scripts/docker_push.sh $(filter-out $@,$(MAKECMDGOALS))

.PHONY: image

%:
	@:
