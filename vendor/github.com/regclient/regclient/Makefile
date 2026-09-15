COMMANDS?=regctl regsync regbot
BINARIES?=$(addprefix bin/,$(COMMANDS))
IMAGES?=$(addprefix docker-,$(COMMANDS))
ARTIFACT_PLATFORMS?=linux-amd64 linux-arm64 linux-ppc64le linux-s390x darwin-amd64 darwin-arm64 windows-amd64.exe
ARTIFACTS?=$(foreach cmd,$(addprefix artifacts/,$(COMMANDS)),$(addprefix $(cmd)-,$(ARTIFACT_PLATFORMS)))
TEST_PLATFORMS?=linux/386,linux/amd64,linux/arm/v6,linux/arm/v7,linux/arm64,linux/ppc64le,linux/s390x
VCS_REPO?="https://github.com/regclient/regclient.git"
VCS_REF?=$(shell git rev-list -1 HEAD)
ifneq ($(shell git status --porcelain 2>/dev/null),)
  VCS_REF := $(VCS_REF)-dirty
endif
VCS_DATE?=$(shell date -d "@$(shell git log -1 --format=%at)" +%Y-%m-%dT%H:%M:%SZ --utc)
VCS_TAG?=$(shell git describe --tags --abbrev=0 2>/dev/null || true)
LD_FLAGS?=-s -w -extldflags -static -buildid= -X \"github.com/regclient/regclient/internal/version.vcsTag=$(VCS_TAG)\"
GO_BUILD_FLAGS?=-trimpath -ldflags "$(LD_FLAGS)" -tags nolegacy
DOCKERFILE_EXT?=$(shell if docker build --help 2>/dev/null | grep -q -- '--progress'; then echo ".buildkit"; fi)
DOCKER_ARGS?=--build-arg "VCS_REF=$(VCS_REF)"
GOPATH?=$(shell go env GOPATH)
PWD:=$(shell pwd)
VER_BUMP?=$(shell command -v version-bump 2>/dev/null)
VER_BUMP_CONTAINER?=sudobmitch/version-bump:edge
ifeq "$(strip $(VER_BUMP))" ''
	VER_BUMP=docker run --rm \
		-v "$(shell pwd)/:$(shell pwd)/" -w "$(shell pwd)" \
		-u "$(shell id -u):$(shell id -g)" \
		$(VER_BUMP_CONTAINER)
endif
SYFT?=$(shell command -v syft 2>/dev/null)
SYFT_CMD_VER:=$(shell [ -x "$(SYFT)" ] && echo "v$$($(SYFT) version | awk '/^Version: / {print $$2}')" || echo "0")
SYFT_VERSION?=v0.77.0
SYFT_CONTAINER?=anchore/syft:v0.77.0@sha256:0135d844712731b86fab20ea584e594ad25b9f5a7fd262b5feb490e865edef89
ifneq "$(SYFT_CMD_VER)" "$(SYFT_VERSION)"
	SYFT=docker run --rm \
		-v "$(shell pwd)/:$(shell pwd)/" -w "$(shell pwd)" \
		-u "$(shell id -u):$(shell id -g)" \
		$(SYFT_CONTAINER)
endif
STATICCHECK_VER?=v0.4.3

.PHONY: all fmt vet test lint lint-go lint-md vendor binaries docker artifacts artifact-pre plugin-user plugin-host .FORCE

.FORCE:

all: fmt vet test lint binaries ## Full build of Go binaries (including fmt, vet, test, and lint)

fmt: ## go fmt
	go fmt ./...

vet: ## go vet
	go vet ./...

test: ## go test
	go test -cover ./...

lint: lint-go lint-md ## Run all linting

lint-go: $(GOPATH)/bin/staticcheck .FORCE ## Run linting for Go
	$(GOPATH)/bin/staticcheck -checks all ./...

lint-md: .FORCE ## Run linting for markdown
	docker run --rm -v "$(PWD):/workdir:ro" ghcr.io/igorshubovych/markdownlint-cli:latest \
	  --ignore vendor .

vendor: ## Vendor Go modules
	go mod vendor

binaries: vendor $(BINARIES) ## Build Go binaries

bin/%: .FORCE
	CGO_ENABLED=0 go build ${GO_BUILD_FLAGS} -o bin/$* ./cmd/$*

docker: $(IMAGES) ## Build Docker images

docker-%: .FORCE
	docker build -t regclient/$* -f build/Dockerfile.$*$(DOCKERFILE_EXT) $(DOCKER_ARGS) .
	docker build -t regclient/$*:alpine -f build/Dockerfile.$*$(DOCKERFILE_EXT) --target release-alpine $(DOCKER_ARGS) .

oci-image: $(addprefix oci-image-,$(COMMANDS)) ## Build reproducible images to an OCI Layout

oci-image-%: bin/regctl .FORCE
	build/oci-image.sh -r scratch -i "$*" -p "$(TEST_PLATFORMS)"
	build/oci-image.sh -r alpine  -i "$*" -p "$(TEST_PLATFORMS)" -b "alpine:3"

test-docker: $(addprefix test-docker-,$(COMMANDS)) ## Build multi-platform docker images (but do not tag)

test-docker-%:
	docker buildx build --platform="$(TEST_PLATFORMS)" -f build/Dockerfile.$*.buildkit .
	docker buildx build --platform="$(TEST_PLATFORMS)" -f build/Dockerfile.$*.buildkit --target release-alpine .

artifacts: $(ARTIFACTS) ## Generate artifacts

artifact-pre:
	mkdir -p artifacts

artifacts/%: artifact-pre .FORCE
	@target="$*"; \
	command="$${target%%-*}"; \
	platform_ext="$${target#*-}"; \
	platform="$${platform_ext%.*}"; \
	export GOOS="$${platform%%-*}"; \
	export GOARCH="$${platform#*-}"; \
	echo export GOOS=$${GOOS}; \
	echo export GOARCH=$${GOARCH}; \
	echo go build ${GO_BUILD_FLAGS} -o "$@" ./cmd/$${command}/; \
	CGO_ENABLED=0 go build ${GO_BUILD_FLAGS} -o "$@" ./cmd/$${command}/; \
	$(SYFT) packages -q "file:$@" --name "$${command}" -o cyclonedx-json >"artifacts/$${command}-$${platform}.cyclonedx.json"; \
	$(SYFT) packages -q "file:$@" --name "$${command}" -o spdx-json >"artifacts/$${command}-$${platform}.spdx.json"

plugin-user:
	mkdir -p ${HOME}/.docker/cli-plugins/
	cp docker-plugin/docker-regclient ${HOME}/.docker/cli-plugins/docker-regctl

plugin-host:
	sudo cp docker-plugin/docker-regclient /usr/libexec/docker/cli-plugins/docker-regctl

util-golang-update:
	go get -u
	go mod tidy
	go mod vendor
	
util-version-check:
	$(VER_BUMP) check

util-version-update:
	$(VER_BUMP) update

$(GOPATH)/bin/staticcheck: .FORCE
	@[ -f $(GOPATH)/bin/staticcheck ] \
	&& [ "$$($(GOPATH)/bin/staticcheck -version | cut -f 3 -d ' ' | tr -d '()')" = "$(STATICCHECK_VER)" ] \
	|| go install "honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VER)"

help: # Display help
	@awk -F ':|##' '/^[^\t].+?:.*?##/ { printf "\033[36m%-30s\033[0m %s\n", $$1, $$NF }' $(MAKEFILE_LIST)
