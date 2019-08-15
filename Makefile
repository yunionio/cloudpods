#####################################################

REPO_PREFIX := yunion.io/x/onecloud
VERSION_PKG := yunion.io/x/pkg/util/version
ROOT_DIR := $(CURDIR)
BUILD_DIR := $(ROOT_DIR)/_output
BIN_DIR := $(BUILD_DIR)/bin
BUILD_SCRIPT := $(ROOT_DIR)/build/build.sh

ifeq ($(ONECLOUD_CI_BUILD),)
  GIT_COMMIT := $(shell git rev-parse --short HEAD)
  GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
  GIT_VERSION := $(shell git describe --tags --abbrev=14 $(GIT_COMMIT)^{commit})
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

GO_BUILD_FLAGS:=-mod vendor -ldflags $(LDFLAGS)
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

export GO111MODULE:=on
export CGO_CFLAGS = ${X_CGO_CFLAGS}
export CGO_LDFLAGS = ${X_CGO_LDFLAGS}

UNAME := $(shell uname)

ifeq ($(UNAME), Linux)
XARGS_FLAGS = --no-run-if-empty
endif

all: build


install: prepare_dir
	@for PKG in $$( $(PKGS) | grep -w "$(filter-out $@,$(MAKECMDGOALS))" ); do \
		echo $$PKG; \
		$(GO_INSTALL) $$PKG; \
	done


build: gendoc
	$(MAKE) $(filter-out cmd/host-image, $(wildcard cmd/*))

gendoc:
	@sh build/gendoc.sh

gencopyright:
	@sh scripts/gencopyright.sh pkg cmd

test:
	@go test $(GO_BUILD_FLAGS) $(shell go list ./... | egrep -v 'host-image|hostimage')

vet:
	go vet ./...

cmd/%: prepare_dir fmt
	$(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@


pkg/%: prepare_dir fmt
	$(GO_INSTALL) $(REPO_PREFIX)/$@


# a hack
rpm:
	$(MAKE) $(patsubst %,cmd/%,$(filter-out $@,$(MAKECMDGOALS)))
	$(foreach cmd,$(filter-out $@,$(MAKECMDGOALS)),$(BUILD_SCRIPT) $(cmd);)

rpmclean:
	rm -fr $(BUILD_DIR)/rpms

prepare_dir: bin_dir


bin_dir: output_dir
	@mkdir -p $(BUILD_DIR)/bin


output_dir:
	@mkdir -p $(BUILD_DIR)


.PHONY: all build prepare_dir clean fmt rpm


clean:
	@rm -fr $(BUILD_DIR)


fmt:
	@$(if $(ONECLOUD_CI_BUILD),:,find) . -type f -name "*.go" -not -path "./_output/*" \
		-not -path "./vendor/*" | xargs $(XARGS_FLAGS) gofmt -s -w

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
	go get $(patsubst %,%@master,$(shell GO111MODULE=on go mod edit -print  | sed -n -e 's|.*\(yunion.io/x/[a-z].*\) v.*|\1|p'))
	go mod tidy
	go mod vendor -v


DOCKER_BUILD_IMAGE_VERSION?=latest

define dockerBuildCmd
set -o errexit
set -o pipefail
cd /home/build/onecloud
export GOFLAGS=-mod=vendor
make $(1)
endef

docker-build: export dockerBuildCmd:=$(call dockerBuildCmd,$(F))
docker-build:
	echo "$$dockerBuildCmd"
	docker rm --force onecloud-ci-build &>/dev/null || true
	docker run \
		--name onecloud-ci-build \
		--rm \
		--volume $(CURDIR):/home/build/onecloud \
		yunionio/onecloud-ci:$(DOCKER_BUILD_IMAGE_VERSION) \
		/bin/bash -c "$$dockerBuildCmd"
	chown -R $$(id -u):$$(id -g) _output
	ls -lh _output/bin

# NOTE we need a way to stop and remove the container started by docker-build.
# No --tty, --stop-signal won't work
docker-build-stop:
	docker stop --time 0 onecloud-ci-build || true

.PHONY: docker-build
.PHONY: docker-build-stop

define helpText
Build with docker

	make docker-build F='-j4'
	make docker-build F='-j4 cmd/region cmd/climc'
	make docker-build-stop

Tidy up go modules and vendor directory

	make mod
endef

help: export helpText:=$(helpText)
help:
	@echo "$$helpText"
