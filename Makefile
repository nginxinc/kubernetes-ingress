all: push

VERSION = 1.7.0-rc1
TAG = $(VERSION)
PREFIX = nginx/nginx-ingress

GOLANG_CONTAINER = golang:1.13
GOFLAGS ?= -mod=vendor
DOCKERFILEPATH = build
DOCKERFILE = Dockerfile # note, this can be overwritten e.g. can be DOCKERFILE=DockerFileForPlus

BUILD_IN_CONTAINER = 1
PUSH_TO_GCR =
GENERATE_DEFAULT_CERT_AND_KEY =
DOCKER_BUILD_OPTIONS =

GIT_COMMIT = $(shell git rev-parse --short HEAD)

export DOCKER_BUILDKIT = 1

lint:
	golangci-lint run

test:
ifneq ($(BUILD_IN_CONTAINER),1)
	GO111MODULE=on GOFLAGS='$(GOFLAGS)' go test ./...
endif

verify-codegen:
ifneq ($(BUILD_IN_CONTAINER),1)
	./hack/verify-codegen.sh
endif

verify-crds:
ifneq ($(BUILD_IN_CONTAINER),1)
	./hack/verify-crds.sh
endif

update-codegen:
	./hack/update-codegen.sh

update-crds:
	./hack/update-crds.sh

certificate-and-key:
ifeq ($(GENERATE_DEFAULT_CERT_AND_KEY),1)
	./build/generate_default_cert_and_key.sh
endif

binary:
ifneq ($(BUILD_IN_CONTAINER),1)
	CGO_ENABLED=0 GO111MODULE=on GOFLAGS='$(GOFLAGS)' GOOS=linux go build -installsuffix cgo -ldflags "-w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT}" -o nginx-ingress github.com/nginxinc/kubernetes-ingress/cmd/nginx-ingress
endif

container: test verify-codegen verify-crds binary certificate-and-key
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
