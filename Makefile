# variables that should not be overridden by the user
VER = $(shell grep IC_VERSION .github/data/version.txt | cut -d '=' -f 2)
GIT_TAG = $(shell git tag --sort=-version:refname | head -n1 || echo untagged)
VERSION = $(VER)-SNAPSHOT
PLUS_ARGS = --secret id=nginx-repo.crt,src=nginx-repo.crt --secret id=nginx-repo.key,src=nginx-repo.key

# Additional flags added here can be accessed in main.go.
# e.g. `main.version` maps to `var version` in main.go
GO_LINKER_FLAGS_VARS = -X main.version=${VERSION}
GO_LINKER_FLAGS_OPTIONS = -s -w
GO_LINKER_FLAGS = $(GO_LINKER_FLAGS_OPTIONS) $(GO_LINKER_FLAGS_VARS)

# variables that can be overridden by the user
PREFIX                        ?= nginx/nginx-ingress ## The name of the image. For example, nginx/nginx-ingress
TAG                           ?= $(VERSION:v%=%) ## The tag of the image. For example, 2.0.0
TARGET                        ?= local ## The target of the build. Possible values: local, container and download
override DOCKER_BUILD_OPTIONS += --build-arg IC_VERSION=$(VERSION) ## The options for the docker build command. For example, --pull
ARCH                          ?= amd64 ## The architecture of the image or binary. For example: amd64, arm64, ppc64le, s390x. Not all architectures are supported for all targets
GOOS                          ?= linux ## The OS of the binary. For example linux, darwin

# final docker build command
DOCKER_CMD = docker build --platform linux/$(strip $(ARCH)) $(strip $(DOCKER_BUILD_OPTIONS)) --target $(strip $(TARGET)) -f build/Dockerfile -t $(strip $(PREFIX)):$(strip $(TAG)) .

export DOCKER_BUILDKIT = 1

.DEFAULT_GOAL:=help

.PHONY: help
help: Makefile ## Display this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "; printf "Usage:\n\n    make \033[36m<target>\033[0m [VARIABLE=value...]\n\nTargets:\n\n"}; {printf "    \033[36m%-30s\033[0m %s\n", $$1, $$2}'
	@grep -E '^(override )?[a-zA-Z_-]+ \??\+?= .*? ## .*$$' $< | sort | awk 'BEGIN {FS = " \\??\\+?= .*? ## "; printf "\nVariables:\n\n"}; {gsub(/override /, "", $$1); printf "    \033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: all
all: test lint verify-codegen update-crds debian-image

.PHONY: lint
lint: ## Run linter
	@git fetch
	docker run --pull always --rm -v $(shell pwd):/kubernetes-ingress -w /kubernetes-ingress -v $(shell go env GOCACHE):/cache/go -e GOCACHE=/cache/go -e GOLANGCI_LINT_CACHE=/cache/go -v $(shell go env GOPATH)/pkg:/go/pkg golangci/golangci-lint:latest git diff -p origin/main > /tmp/diff.patch && golangci-lint --color always run -v --new-from-patch=/tmp/diff.patch

.PHONY: lint-python
lint-python: ## Run linter for python tests
	@isort -V || (code=$$?; printf "\033[0;31mError\033[0m: there was a problem with isort, use 'brew install isort' to install it\n"; exit $$code)
	@black --version || (code=$$?; printf "\033[0;31mError\033[0m: there was a problem with black, use 'brew install black' to install it\n"; exit $$code)
	@isort .
	@black .

.PHONY: staticcheck
staticcheck: ## Run staticcheck linter
	@staticcheck -version >/dev/null 2>&1 || go install honnef.co/go/tools/cmd/staticcheck@2022.1.3;
	staticcheck ./...

.PHONY: test
test: ## Run GoLang tests
	go test -tags=aws -shuffle=on -race ./...

cover: ## Generate coverage report
	@./hack/test-cover.sh

cover-html: ## Generate and show coverage report in HTML format
	go test -tags=aws -shuffle=on -race ./... -count=1 -cover -covermode=atomic -coverprofile=coverage.out
	go tool cover -html coverage.out

.PHONY: verify-codegen
verify-codegen: ## Verify code generation
	./hack/verify-codegen.sh

.PHONY: update-codegen
update-codegen: ## Generate code
	./hack/update-codegen.sh

.PHONY: update-crds
update-crds: ## Update CRDs
	go run sigs.k8s.io/controller-tools/cmd/controller-gen crd paths=./pkg/apis/... output:crd:artifacts:config=config/crd/bases
	@kustomize version || (code=$$?; printf "\033[0;31mError\033[0m: there was a problem with kustomize, use 'brew install kustomize' to install it\n"; exit $$code)
	kustomize build config/crd >deploy/crds.yaml
	kustomize build config/crd/app-protect-dos --load-restrictor='LoadRestrictionsNone' >deploy/crds-nap-dos.yaml
	kustomize build config/crd/app-protect-waf --load-restrictor='LoadRestrictionsNone' >deploy/crds-nap-waf.yaml

.PHONY: certificate-and-key
certificate-and-key: ## Create default cert and key
	./build/generate_default_cert_and_key.sh

.PHONY: build
build: ## Build Ingress Controller binary
	@docker -v || (code=$$?; printf "\033[0;31mError\033[0m: there was a problem with Docker\n"; exit $$code)
ifeq ($(strip $(TARGET)),local)
	@go version || (code=$$?; printf "\033[0;31mError\033[0m: unable to build locally, try using the parameter TARGET=container or TARGET=download\n"; exit $$code)
	CGO_ENABLED=0 GOOS=$(strip $(GOOS)) GOARCH=$(strip $(ARCH)) go build -trimpath -ldflags "$(GO_LINKER_FLAGS)" -o nginx-ingress github.com/nginxinc/kubernetes-ingress/cmd/nginx-ingress
else ifeq ($(strip $(TARGET)),download)
	@$(MAKE) download-binary-docker
endif

.PHONY: download-binary-docker
download-binary-docker: ## Download Docker image from which to extract Ingress Controller binary, TARGET=download is required
ifeq ($(strip $(TARGET)),download)
DOWNLOAD_TAG := $(shell ./hack/docker.sh $(GIT_TAG))
ifeq ($(DOWNLOAD_TAG),fail)
$(error unable to build with TARGET=download, this function is only available when building from a git tag or from the latest commit matching the edge image)
endif
override DOCKER_BUILD_OPTIONS += --build-arg DOWNLOAD_TAG=$(DOWNLOAD_TAG)
endif

.PHONY: build-goreleaser
build-goreleaser: ## Build Ingress Controller binary using GoReleaser
	@goreleaser -v || (code=$$?; printf "\033[0;31mError\033[0m: there was a problem with GoReleaser. Follow the docs to install it https://goreleaser.com/install\n"; exit $$code)
	GOOS=linux GOPATH=$(shell go env GOPATH) GOARCH=$(strip $(ARCH)) goreleaser build --clean --debug --snapshot --id kubernetes-ingress --single-target

.PHONY: debian-image
debian-image: build ## Create Docker image for Ingress Controller (Debian)
	$(DOCKER_CMD) --build-arg BUILD_OS=debian

.PHONY: alpine-image
alpine-image: build ## Create Docker image for Ingress Controller (Alpine)
	$(DOCKER_CMD) --build-arg BUILD_OS=alpine

.PHONY: alpine-image-plus
alpine-image-plus: build ## Create Docker image for Ingress Controller (Alpine with NGINX Plus)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=alpine-plus

.PHONY: alpine-image-plus-fips
alpine-image-plus-fips: build ## Create Docker image for Ingress Controller (Alpine with NGINX Plus and FIPS)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=alpine-plus-fips

.PHONY: alpine-image-nap-plus-fips
alpine-image-nap-plus-fips: build ## Create Docker image for Ingress Controller (Alpine with NGINX Plus, NGINX App Protect WAF and FIPS)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=alpine-plus-nap-waf-fips

.PHONY: debian-image-plus
debian-image-plus: build ## Create Docker image for Ingress Controller (Debian with NGINX Plus)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=debian-plus

.PHONY: debian-image-nap-plus
debian-image-nap-plus: build ## Create Docker image for Ingress Controller (Debian with NGINX Plus and NGINX App Protect WAF)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=debian-plus-nap --build-arg NAP_MODULES=waf

.PHONY: debian-image-dos-plus
debian-image-dos-plus: build ## Create Docker image for Ingress Controller (Debian with NGINX Plus and NGINX App Protect DoS)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=debian-plus-nap --build-arg NAP_MODULES=dos

.PHONY: debian-image-nap-dos-plus
debian-image-nap-dos-plus: build ## Create Docker image for Ingress Controller (Debian with NGINX Plus, NGINX App Protect WAF and DoS)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=debian-plus-nap --build-arg NAP_MODULES=waf,dos

.PHONY: ubi-image
ubi-image: build ## Create Docker image for Ingress Controller (UBI)
	$(DOCKER_CMD) --build-arg BUILD_OS=ubi

.PHONY: ubi-image-plus
ubi-image-plus: build ## Create Docker image for Ingress Controller (UBI with NGINX Plus)
	$(DOCKER_CMD) $(PLUS_ARGS) --build-arg BUILD_OS=ubi-plus

.PHONY: ubi-image-nap-plus
ubi-image-nap-plus: build ## Create Docker image for Ingress Controller (UBI with NGINX Plus and NGINX App Protect WAF)
	$(DOCKER_CMD) $(PLUS_ARGS) --secret id=rhel_license,src=rhel_license --build-arg BUILD_OS=ubi-9-plus-nap --build-arg NAP_MODULES=waf

.PHONY: ubi-image-dos-plus
ubi-image-dos-plus: build ## Create Docker image for Ingress Controller (UBI with NGINX Plus and NGINX App Protect DoS)
	$(DOCKER_CMD) $(PLUS_ARGS) --secret id=rhel_license,src=rhel_license --build-arg BUILD_OS=ubi-8-plus-nap --build-arg NAP_MODULES=dos

.PHONY: ubi-image-nap-dos-plus
ubi-image-nap-dos-plus: build ## Create Docker image for Ingress Controller (UBI with NGINX Plus, NGINX App Protect WAF and DoS)
	$(DOCKER_CMD) $(PLUS_ARGS) --secret id=rhel_license,src=rhel_license --build-arg BUILD_OS=ubi-8-plus-nap --build-arg NAP_MODULES=waf,dos

.PHONY: all-images ## Create all the Docker images for Ingress Controller
all-images: alpine-image alpine-image-plus debian-image debian-image-plus debian-image-nap-plus debian-image-dos-plus debian-image-nap-dos-plus ubi-image ubi-image-plus ubi-image-nap-plus ubi-image-dos-plus ubi-image-nap-dos-plus

.PHONY: push
push: ## Docker push to PREFIX and TAG
	docker push $(strip $(PREFIX)):$(strip $(TAG))

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

.PHONY: rebuild-test-img ## Rebuild the python e2e test image
rebuild-test-img:
	cd tests && \
	make build
