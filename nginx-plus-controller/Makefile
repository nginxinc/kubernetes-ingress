all: push

VERSION = 0.7.0
TAG = $(VERSION)
PREFIX =

DOCKER_RUN = docker run --rm -v $(shell pwd)/../:/go/src/github.com/nginxinc/kubernetes-ingress -w /go/src/github.com/nginxinc/kubernetes-ingress/nginx-plus-controller/
GOLANG_CONTAINER = golang:1.7

BUILD_IN_CONTAINER = 1
PUSH_TO_GCR =

ifeq ($(PUSH_TO_GCR),1)
GCLOUD = gcloud
endif

nginx-plus-ingress:
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) -e CGO_ENABLED=0 $(GOLANG_CONTAINER) go build -a -installsuffix cgo -ldflags "-w -X main.version=${VERSION}" -o nginx-plus-ingress *.go
else
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-w -X main.version=${VERSION}" -o nginx-plus-ingress *.go
endif

test:
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) $(GOLANG_CONTAINER) go test ./...
else
	go test ./...
endif

container: test nginx-plus-ingress
	docker build -t $(PREFIX):$(TAG) .

push: container
ifeq ($(PUSH_TO_GCR),1)
	$(GCLOUD) docker -- push $(PREFIX):$(TAG)
else
	docker push $(PREFIX):$(TAG)
endif

osx:
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) -e CGO_ENABLED=0 -e GOOS=darwin $(GOLANG_CONTAINER) go build -a -installsuffix cgo -ldflags "-w -X main.version=${VERSION}" -o osx-nginx-plus-ingress *.go
else
	CGO_ENABLED=0 GOOS=darwin go build -a -installsuffix cgo -ldflags "-w -X main.version=${VERSION}" -o osx-nginx-plus-ingress *.go
endif

clean:
	rm -f nginx-plus-ingress

