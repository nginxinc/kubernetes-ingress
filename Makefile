all: push

VERSION = 1.10.1
TAG = $(VERSION)
PREFIX = nginx/nginx-ingress

GOLANG_CONTAINER = golang:1.15
GOFLAGS ?= -mod=vendor
DOCKERFILEPATH = build
DOCKERFILE = Dockerfile # note, this can be overwritten e.g. can be DOCKERFILE=DockerFileForPlus

BUILD_IN_CONTAINER = 1
PUSH_TO_GCR =
GENERATE_DEFAULT_CERT_AND_KEY =

GIT_COMMIT = $(shell git rev-parse --short HEAD)

export DOCKER_BUILDKIT = 1

lint:
	golangci-lint run

test:
ifneq ($(BUILD_IN_CONTAINER),1)
	@go version || (code=$$?; printf "\033[0;31mError\033[0m: unable to build locally, try using the parameter BUILD_IN_CONTAINER=1\n"; exit $$code)
	GO111MODULE=on GOFLAGS='$(GOFLAGS)' go test ./...
endif

verify-codegen:
ifneq ($(BUILD_IN_CONTAINER),1)
	./hack/verify-codegen.sh
endif

update-codegen:
	./hack/update-codegen.sh

update-crds:
ifneq ($(BUILD_IN_CONTAINER),1)
	go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:crdVersions=v1 schemapatch:manifests=./deployments/common/crds/ paths=./pkg/apis/configuration/... output:dir=./deployments/common/crds
	go run sigs.k8s.io/controller-tools/cmd/controller-gen crd:crdVersions=v1beta1,preserveUnknownFields=false schemapatch:manifests=./deployments/common/crds-v1beta1/ paths=./pkg/apis/configuration/... output:dir=./deployments/common/crds-v1beta1
	@cp -Rp deployments/common/crds-v1beta1/ deployments/helm-chart/crds
endif

certificate-and-key:
ifeq ($(GENERATE_DEFAULT_CERT_AND_KEY),1)
	./build/generate_default_cert_and_key.sh
endif

binary:
ifneq ($(BUILD_IN_CONTAINER),1)
	CGO_ENABLED=0 GO111MODULE=on GOFLAGS='$(GOFLAGS)' GOOS=linux go build -installsuffix cgo -ldflags "-w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT}" -o nginx-ingress github.com/nginxinc/kubernetes-ingress/cmd/nginx-ingress
endif

prepare-options-secrets:
ifneq (,$(findstring Plus,$(DOCKERFILE)))
override DOCKER_BUILD_OPTIONS += --secret id=nginx-repo.crt,src=nginx-repo.crt --secret id=nginx-repo.key,src=nginx-repo.key
endif
ifneq (,$(findstring PlusForOpenShift,$(DOCKERFILE)))
override DOCKER_BUILD_OPTIONS += --secret id=rhel_license,src=rhel_license
endif

container: test verify-codegen update-crds binary certificate-and-key prepare-options-secrets
	@docker -v || (code=$$?; printf "\033[0;31mError\033[0m: there was a problem with Docker\n"; exit $$code)
ifeq ($(BUILD_IN_CONTAINER),1)
	docker build $(DOCKER_BUILD_OPTIONS) --build-arg IC_VERSION=$(VERSION)-$(GIT_COMMIT) --build-arg GIT_COMMIT=$(GIT_COMMIT) --build-arg VERSION=$(VERSION) --build-arg GOLANG_CONTAINER=$(GOLANG_CONTAINER) --target container -f $(DOCKERFILEPATH)/$(DOCKERFILE) -t $(PREFIX):$(TAG) .
else
	docker build $(DOCKER_BUILD_OPTIONS) --build-arg IC_VERSION=$(VERSION)-$(GIT_COMMIT) --target local -f $(DOCKERFILEPATH)/$(DOCKERFILE) -t $(PREFIX):$(TAG) .
endif

push: container
ifeq ($(PUSH_TO_GCR),1)
	gcloud docker -- push $(PREFIX):$(TAG)
else
	docker push $(PREFIX):$(TAG)
endif

clean:
	rm -f nginx-ingress
