all: push

VERSION = 1.6.2
TAG = $(VERSION)
PREFIX = nginx/nginx-ingress

DOCKER_TEST_RUN = docker run --rm -v $(shell pwd):/go/src/github.com/nginxinc/kubernetes-ingress -w /go/src/github.com/nginxinc/kubernetes-ingress
DOCKER_BUILD_RUN = docker run --rm -v $(shell pwd):/go/src/github.com/nginxinc/kubernetes-ingress -w /go/src/github.com/nginxinc/kubernetes-ingress/cmd/nginx-ingress/
GOLANG_CONTAINER = golang:1.13
GOFLAGS ?= -mod=vendor
DOCKERFILEPATH = build
DOCKERFILE = Dockerfile # note, this can be overwritten e.g. can be DOCKERFILE=DockerFileForPlus

BUILD_IN_CONTAINER = 1
PUSH_TO_GCR =
GENERATE_DEFAULT_CERT_AND_KEY =
DOCKER_BUILD_OPTIONS =

GIT_COMMIT=$(shell git rev-parse --short HEAD)

nginx-ingress:
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_BUILD_RUN) -e CGO_ENABLED=0 -e GO111MODULE=on -e GOFLAGS='$(GOFLAGS)' $(GOLANG_CONTAINER) go build -installsuffix cgo -ldflags "-w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT}" -o /go/src/github.com/nginxinc/kubernetes-ingress/nginx-ingress
else
	CGO_ENABLED=0 GO111MODULE=on GOFLAGS='$(GOFLAGS)' GOOS=linux go build -installsuffix cgo -ldflags "-w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT}" -o nginx-ingress github.com/nginxinc/kubernetes-ingress/cmd/nginx-ingress
endif

lint:
	golangci-lint run

test:
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_TEST_RUN) -e GO111MODULE=on -e GOFLAGS='$(GOFLAGS)' $(GOLANG_CONTAINER) go test ./...
else
	GO111MODULE=on GOFLAGS='$(GOFLAGS)' go test ./...
endif

verify-codegen:
ifneq ($(BUILD_IN_CONTAINER), 1)
	./hack/verify-codegen.sh
endif

update-codegen:
	./hack/update-codegen.sh

certificate-and-key:
ifeq ($(GENERATE_DEFAULT_CERT_AND_KEY),1)
	./build/generate_default_cert_and_key.sh
endif

container: test verify-codegen nginx-ingress certificate-and-key
	cp $(DOCKERFILEPATH)/$(DOCKERFILE) ./Dockerfile
	docker build $(DOCKER_BUILD_OPTIONS) --build-arg IC_VERSION=$(VERSION)-$(GIT_COMMIT) -f Dockerfile -t $(PREFIX):$(TAG) .

push: container
ifeq ($(PUSH_TO_GCR),1)
	gcloud docker -- push $(PREFIX):$(TAG)
else
	docker push $(PREFIX):$(TAG)
endif

clean:
	rm -f nginx-ingress
	rm -f Dockerfile
