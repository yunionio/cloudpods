#####################################################

REPO_PREFIX := yunion.io/x/onecloud
VENDOR_PATH := $(REPO_PREFIX)/vendor
VERSION_PKG := $(VENDOR_PATH)/yunion.io/x/pkg/util/version
ROOT_DIR := $(shell pwd)
BUILD_DIR := $(ROOT_DIR)/_output
BIN_DIR := $(BUILD_DIR)/bin
BUILD_SCRIPT := $(ROOT_DIR)/build/build.sh

GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_VERSION := $(shell git describe --tags --abbrev=14 $(GIT_COMMIT)^{commit})
GIT_TREE_STATE := $(shell s=`git status --porcelain 2>/dev/null`; if [ -z "$$s" ]; then echo "clean"; else echo "dirty"; fi)
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := "-w \
	-X $(VERSION_PKG).gitVersion=$(GIT_VERSION) \
	-X $(VERSION_PKG).gitCommit=$(GIT_COMMIT) \
	-X $(VERSION_PKG).gitBranch=$(GIT_BRANCH) \
	-X $(VERSION_PKG).buildDate=$(BUILD_DATE) \
	-X $(VERSION_PKG).gitTreeState=$(GIT_TREE_STATE) \
	-X $(VERSION_PKG).gitMajor=0 \
	-X $(VERSION_PKG).gitMinor=0"


#####################################################

GO_BUILD := go build -ldflags $(LDFLAGS)
GO_INSTALL := go install -ldflags $(LDFLAGS)
GO_TEST := go test

PKGS := go list ./...
CMDS := $(shell find ./cmd -mindepth 1 -maxdepth 1 -type d)


all: build


install: prepare_dir
	@for PKG in $$( $(PKGS) | grep -w "$(filter-out $@,$(MAKECMDGOALS))" ); do \
		echo $$PKG; \
		$(GO_INSTALL) $$PKG; \
	done


build: prepare_dir
	@for PKG in $(CMDS); do \
		echo build $$PKG; \
		$(GO_BUILD) -o $(BIN_DIR)/`basename $${PKG}` $$PKG; \
	done


test: prepare_dir
	@for PKG in $$( $(PKGS) | grep "$(filter-out $@,$(MAKECMDGOALS))" ); do \
		echo $$PKG; \
		$(GO_TEST) $$PKG; \
	done


cmd/%: prepare_dir
	$(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@


pkg/%: prepare_dir
	$(GO_INSTALL) $(REPO_PREFIX)/$@


rpm:
	make cmd/$(filter-out $@,$(MAKECMDGOALS))
	$(BUILD_SCRIPT) $(filter-out $@,$(MAKECMDGOALS))

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
	find . -type f -name "*.go" -not -path "./_output/*" \
		-not -path "./vendor/*" | xargs gofmt -s -w

%:
	@:
