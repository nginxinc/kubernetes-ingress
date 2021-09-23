# variables that should not be overridden by the user
GIT_COMMIT = $(shell git rev-parse HEAD || echo unknown)
GIT_COMMIT_SHORT = $(shell echo ${GIT_COMMIT} | cut -c1-7)
GIT_TAG = $(shell git describe --tags --abbrev=0 || echo untagged)
DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
VERSION = $(GIT_TAG)-SNAPSHOT-$(GIT_COMMIT_SHORT)
PLUS_ARGS = --secret id=nginx-repo.crt,src=nginx-repo.crt --secret id=nginx-repo.key,src=nginx-repo.key

# variables that can be overridden by the user
PREFIX = nginx/nginx-ingress ## The name of the image. For example, nginx/nginx-ingress
TAG = $(VERSION:v%=%) ## The tag of the image. For example, 2.0.0
TARGET ?= local ## The target of the build. Possible values: local, container and download
override DOCKER_BUILD_OPTIONS += --build-arg IC_VERSION=$(VERSION) --build-arg GIT_COMMIT=$(GIT_COMMIT) --build-arg DATE=$(DATE) ## The options for the docker build command. For example, --pull.

# final docker build command
DOCKER_CMD = docker build $(strip $(DOCKER_BUILD_OPTIONS)) --target $(strip $(TARGET)) -f build/Dockerfile -t $(strip $(PREFIX)):$(strip $(TAG)) .

export DOCKER_BUILDKIT = 1

.DEFAULT_GOAL:=help

.PHONY: help
help: Makefile ## Display this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "; printf "Usage:\n\n    make \033[36m<target>\033[0m [VARIABLE=value...]\n\nTargets:\n\n"}; {printf "    \033[36m%-30s\033[0m %s\n", $$1, $$2}'
	@grep -E '^(override )?[a-zA-Z_-]+ \??\+?= .*?## .*$$' $< | sort | awk 'BEGIN {FS = " \\??\\+?= .*?## "; printf "\nVariables:\n\n"}; {gsub(/override /, "", $$1); printf "    \033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: all
all: test lint verify-codegen update-crds debian-image

.PHONY: lint
lint: ## Run linter
	docker run --pull always --rm -v $(shell pwd):/kubernetes-ingress -w /kubernetes-ingress -v $(shell go env GOCACHE):/cache/go -e GOCACHE=/cache/go -e GOLANGCI_LINT_CACHE=/cache/go -v $(shell go env GOPATH)/pkg:/go/pkg golangci/golangci-lint:latest golangci-lint --color always run -v

.PHONY: test
test: ## Run tests
	GO111MODULE=on go test ./...

.PHONY: verify-codegen
verify-codegen: ## Verify code generation
	./hack/verify-codegen.sh

.PHONY: update-codegen
update-codegen: ## Generate code
	./hack/update-codegen.sh

.PHONY: update-crds
update-crds: ## Update CRDs
	go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:crdVersions=v1 schemapatch:manifests=./deployments/common/crds/ paths=./pkg/apis/.../... output:dir=./deployments/common/crds
	@cp -Rp deployments/common/crds/ deployments/helm-chart/crds

.PHONY: certificate-and-key
certificate-and-key: ## Create default cert and key
	./build/generate_default_cert_and_key.sh

.PHONY: build
build: ## Build Ingress Controller binary
	@docker -v || (code=$$?; printf "\033[0;31mError\033[0m: there was a problem with Docker\n"; exit $$code)
ifeq (${TARGET},local)
	@go version || (code=$$?; printf "\033[0;31mError\033[0m: unable to build locally, try using the parameter TARGET=container or TARGET=download\n"; exit $$code)
	CGO_ENABLED=0 GO111MODULE=on GOOS=linux go build -trimpath -ldflags "-s -w -X main.version=${VERSION} -X main.commit=${GIT_COMMIT} -X main.date=$(DATE)" -o nginx-ingress github.com/nginxinc/kubernetes-ingress/cmd/nginx-ingress
else ifeq (${TARGET},download)
	@$(MAKE) download-binary-docker
endif

.PHONY: download-binary-docker
download-binary-docker: ## Download Docker image from which to extract Ingress Controller binary, TARGET=download is required
ifeq (${TARGET},download)
DOWNLOAD_TAG := $(shell ./hack/docker.sh $(GIT_COMMIT) $(GIT_TAG))
ifeq ($(DOWNLOAD_TAG),fail)
$(error unable to build with TARGET=download, this function is only available when building from a git tag or from the latest commit matching the edge image)
endif
override DOCKER_BUILD_OPTIONS += --build-arg DOWNLOAD_TAG=$(DOWNLOAD_TAG)
endif

.PHONY: build-goreleaser
build-goreleaser: ## Build Ingress Controller binary using GoReleaser
	@goreleaser -v || (code=$$?; printf "\033[0;31mError\033[0m: there was a problem with GoReleaser. Follow the docs to install it https://goreleaser.com/install\n"; exit $$code)
	GOPATH=$(shell go env GOPATH) goreleaser build --rm-dist --debug --snapshot --id kubernetes-ingress

.PHONY: debian-image
debian-image: build ## Create Docker image for Ingress Controller (Debian)
	$(DOCKER_CMD) --build-arg BUILD_OS=debian

.PHONY: alpine-image
alpine-image: build ## Create Docker image for Ingress Controller (Alpine)
	$(DOCKER_CMD) --build-arg BUILD_OS=alpine

.PHONY: alpine-image-plus
alpine-image-plus: build ## Create Docker image for Ingress Controller (Alpine with NGINX Plus)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=alpine-plus

.PHONY: debian-image-plus
debian-image-plus: build ## Create Docker image for Ingress Controller (Debian with NGINX Plus)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=debian-plus

.PHONY: debian-image-nap-plus
debian-image-nap-plus: build ## Create Docker image for Ingress Controller (Debian with NGINX Plus and App Protect WAF)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=debian-plus-nap --build-arg DEBIAN_VERSION=buster-slim

.PHONY: debian-image-dos-plus
debian-image-dos-plus: build ## Create Docker image for Ingress Controller (nginx plus with nap-dos)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=debian-plus-dos

.PHONY: debian-image-nap-dos-plus
debian-image-nap-dos-plus: build ## Create Docker image for Ingress Controller (nginx plus with nap and nap-dos)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=debian-plus-nap-dos --build-arg FILES=nap-common

.PHONY: openshift-image
openshift-image: build ## Create Docker image for Ingress Controller (UBI)
	$(DOCKER_CMD) --build-arg BUILD_OS=ubi

.PHONY: openshift-image-plus
openshift-image-plus: build ## Create Docker image for Ingress Controller (UBI with NGINX Plus)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=ubi-plus

.PHONY: openshift-image-nap-plus
openshift-image-nap-plus: build ## Create Docker image for Ingress Controller (UBI with NGINX Plus and App Protect WAF)
	$(DOCKER_CMD) $(PLUS_ARGS) --secret id=rhel_license,src=rhel_license --build-arg BUILD_OS=ubi-plus-nap --build-arg UBI_VERSION=7

.PHONY: alpine-image-opentracing
alpine-image-opentracing: build ## Create Docker image for Ingress Controller (Alpine with OpenTracing)
	$(DOCKER_CMD) --build-arg BUILD_OS=alpine-opentracing

.PHONY: openshift-image-dos-plus
openshift-image-dos-plus: build ## Create Docker image for Ingress Controller (ubi with plus and dos)
	$(DOCKER_CMD) $(PLUS_ARGS) $(NAP_ARGS) --secret id=rhel_license,src=rhel_license --build-arg BUILD_OS=ubi-plus-dos --build-arg UBI_VERSION=7

.PHONY: openshift-image-nap-dos-plus
openshift-image-nap-dos-plus: build ## Create Docker image for Ingress Controller (ubi with plus, nap and dos)
	$(DOCKER_CMD) $(PLUS_ARGS) $(NAP_ARGS) --secret id=rhel_license,src=rhel_license --build-arg BUILD_OS=ubi-plus-nap-dos --build-arg UBI_VERSION=7

.PHONY: debian-image-opentracing
debian-image-opentracing: build ## Create Docker image for Ingress Controller (Debian with OpenTracing)
	$(DOCKER_CMD) --build-arg BUILD_OS=opentracing

.PHONY: debian-image-opentracing-plus
debian-image-opentracing-plus: build ## Create Docker image for Ingress Controller (Debian with OpenTracing and NGINX Plus)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=opentracing-plus

.PHONY: all-images ## Create all the Docker images for Ingress Controller
all-images: alpine-image alpine-image-plus debian-image debian-image-plus debian-image-nap-plus debian-image-dos-plus debian-image-nap-dos-plus debian-image-opentracing debian-image-opentracing-plus openshift-image openshift-image-plus openshift-image-nap-plus

.PHONY: push
push: ## Docker push to PREFIX and TAG
	docker push $(PREFIX):$(TAG)

.PHONY: clean
clean:  ## Remove nginx-ingress binary
	-rm nginx-ingress
	-rm -r dist

.PHONY: deps
deps: ## Add missing and remove unused modules, verify deps and download them to local cache
	@go mod tidy && go mod verify && go mod download

.PHONY: clean-cache
clean-cache: ## Clean go cache
	@go clean -modcache
