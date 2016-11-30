all: push

VERSION = 0.6.3-merged
TAG = $(VERSION)
PREFIX = quay.io/nico_schieder/nginxinc-kubernetes-ingress

DOCKER_RUN = docker run --rm -v $(shell pwd)/../:/go/src/github.com/nginxinc/kubernetes-ingress:Z -w /go/src/github.com/nginxinc/kubernetes-ingress/nginx-controller/
GOLANG_CONTAINER = golang:1.6
DOCKERFILE = Dockerfile

BUILD_IN_CONTAINER = 0
PUSH_TO_GCR =

ifeq ($(PUSH_TO_GCR),1)
GCLOUD = gcloud
endif

nginx-ingress:
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) -e CGO_ENABLED=0 $(GOLANG_CONTAINER) go build -a -installsuffix cgo -ldflags "-w -X main.version=${VERSION}" -o nginx-ingress *.go
else
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-w -X main.version=${VERSION}" -o nginx-ingress *.go
endif

test:
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) $(GOLANG_CONTAINER) go test ./...
else
	go test ./...
endif

container: test nginx-ingress
	docker build -f $(DOCKERFILE) -t $(PREFIX):$(TAG) .

push: container
	$(GCLOUD) docker push $(PREFIX):$(TAG)

osx:
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) -e CGO_ENABLED=0 -e GOOS=darwin $(GOLANG_CONTAINER) go build -a -installsuffix cgo -ldflags "-w -X main.version=${VERSION}" -o osx-nginx-ingress *.go
else
	CGO_ENABLED=0 GOOS=darwin go build -a -installsuffix cgo -ldflags "-w -X main.version=${VERSION}" -o osx-nginx-ingress *.go
endif

clean:
	rm -f nginx-ingress
