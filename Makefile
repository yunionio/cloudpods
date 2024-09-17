#####################################################

REPO_PREFIX := yunion.io/x/onecloud
VERSION_PKG := yunion.io/x/pkg/util/version
ROOT_DIR := $(CURDIR)
BUILD_DIR := $(ROOT_DIR)/_output
BIN_DIR := $(BUILD_DIR)/bin
BUILD_SCRIPT := $(ROOT_DIR)/build/build.sh
DEB_BUILD_SCRIPT := $(ROOT_DIR)/build/build_deb.sh

ifeq ($(ONECLOUD_CI_BUILD),)
	GIT_COMMIT := $(shell git rev-parse --short HEAD)
	GIT_BRANCH := $(shell git branch -r --contains | head -1 | sed -E -e "s%(HEAD ->|origin|upstream)/?%%g" | xargs)
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
		X_CGO_LDFLAGS := ${CGO_LDFLAGS_ENV} -laio -lqemuio -lpthread -lgnutls -lnettle -L ${LIBQEMUIO_PATH}/src
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
debTargets:=$(foreach b,$(patsubst cmd/%,%,$(cmdTargets)),$(if $(shell [ -f "$(CURDIR)/build/$(b)/vars" ] && echo 1),deb/$(b)))

all: build


install: prepare_dir
	@for PKG in $$( $(PKGS) | grep -w "$(filter-out $@,$(MAKECMDGOALS))" ); do \
		echo $$PKG; \
		$(GO_INSTALL) $$PKG; \
	done


gencopyright:
	@bash scripts/gencopyright.sh pkg cmd

test:
	@go test $(GO_BUILD_FLAGS) $(shell go list ./... | egrep -v 'host-image|hostimage|torrent')

vet:
	go vet ./...

# cmd/esxi-agent: prepare_dir
# 	CGO_ENABLED=0 $(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@

cmd/host: prepare_dir
	CGO_ENABLED=1 $(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@

cmd/host-image: prepare_dir
	CGO_ENABLED=1 $(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@

cmd/%: prepare_dir
	CGO_ENABLED=0 $(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@

rpm/%: cmd/%
	$(BUILD_SCRIPT) $*

deb/%: cmd/%
	$(DEB_BUILD_SCRIPT) $*

pkg/%: prepare_dir
	$(GO_INSTALL) $(REPO_PREFIX)/$@

rpm/fetcherfs: cmd/fetcherfs
	docker run --rm \
		--name docker-centos-build-fetcherfs \
		-v $(CURDIR):/data \
		registry.cn-beijing.aliyuncs.com/yunionio/centos-build:1.1-4 \
		/bin/bash -c "VERSION=3.6 /data/build/build.sh fetcherfs /opt/yunion/fetchclient/bin"

deb/fetcherfs: rpm/fetcherfs
	#VERSION=3.6 $(DEB_BUILD_SCRIPT) fetcherfs /opt/yunion/fetchclient/bin
	docker run --rm \
		--name docker-debian-build-fetcherfs \
		-v $(CURDIR):/data \
		registry.cn-beijing.aliyuncs.com/yunionio/debian10-base:1.0 \
		/data/build/convert_rpm2deb.sh

build:
	$(MAKE) $(cmdTargets)

rpm:
	$(MAKE) $(rpmTargets)

deb:
	$(MAKE) $(debTargets)

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
		| while read f; do \
			if ! grep -m1 -q '^// Code generated .* DO NOT EDIT\.$$' "$$f"; then \
				echo "$$f"; \
			fi ; \
		done \
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

comma:=,
space:=$(space) $(space)
# NOTE: keep y18n-packages in alphabetical order
y18n-src-lang := en-US
y18n-lang     := en-US,zh-CN
y18n-packages := \
		yunion.io/x/onecloud/cmd/apigateway \
		yunion.io/x/onecloud/cmd/keystone \
		yunion.io/x/onecloud/cmd/monitor \
		yunion.io/x/onecloud/cmd/region \
		yunion.io/x/onecloud/cmd/yunionconf \

define y18n-gen
	set -o errexit; \
	set -o pipefail; \
	export GO111MODULE=off; \
	y18n \
		-chdir $(CURDIR) \
		-dir ./locales/ \
		-out ./locales/locales.go \
		-lang $(y18n-lang) \
		$(y18n-packages) \
		; \
	$(foreach lang,$(filter-out $(y18n-src-lang),$(subst $(comma), ,$(y18n-lang))),cp ./locales/$(lang)/{out,messages}.gotext.json;) \

endef

y18n-gen:
	$(y18n-gen)
	$(y18n-gen)

.PHONY: y18n-gen

y18n-check:
	$(y18n-gen)
	if git status --short ./locales | sed 's/^/$@: /' | grep .; then \
		echo "$@: Locales content needs care" >&2 ; \
		false; \
	fi

.PHONY: y18n-check

define hostdeployer-grpc-gen
	set -o errexit; \
	set -o pipefail; \
	protoc -I pkg/hostman/hostdeployer/apis \
		--go_out=pkg/hostman/hostdeployer/apis \
		--go_opt=paths=source_relative \
		--go-grpc_out=pkg/hostman/hostdeployer/apis \
		--go-grpc_opt=paths=source_relative \
		pkg/hostman/hostdeployer/apis/deploy.proto

endef

hostdeployer-grpc-gen:
	$(hostdeployer-grpc-gen)

check: fmt-check
check: gendocgo-check
check: goimports-check
#check: vet-check
#check: y18n-check
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

RELEASE_BRANCH:=master
GOPROXY ?= direct

mod:
	GOPROXY=$(GOPROXY) GONOSUMDB=yunion.io/x go get -d yunion.io/x/cloudmux@$(RELEASE_BRANCH)
	GOPROXY=$(GOPROXY) GONOSUMDB=yunion.io/x go get -d $(patsubst %,%@master,$(shell GO111MODULE=on go mod edit -print  | sed -n -e 's|.*\(yunion.io/x/[a-z].*\) v.*|\1|p' | grep -v '/cloudmux$$'))
	GOPROXY=$(GOPROXY) GONOSUMDB=yunion.io/x go mod tidy
	GOPROXY=$(GOPROXY) GONOSUMDB=yunion.io/x go mod vendor -v

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

image:
	mkdir -p $(ROOT_DIR)/_output
	DEBUG=$(DEBUG) ARCH=$(ARCH) TAG=$(VERSION) REGISTRY=$(REGISTRY) $(ROOT_DIR)/scripts/docker_push.sh $(filter-out $@,$(MAKECMDGOALS))

.PHONY: image

image-telegraf-raid-plugin:
	VERSION=release-1.6.5 ARCH=all make image telegraf-raid-plugin

%:
	@:

ModName:=yunion.io/x/onecloud
include $(CURDIR)/Makefile.common.mk
