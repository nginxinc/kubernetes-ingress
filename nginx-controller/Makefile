all: push

VERSION = 1.0.0
TAG = $(VERSION)
PREFIX = nginxdemos/nginx-ingress

DOCKER_RUN = docker run --rm -v $(shell pwd)/../:/go/src/github.com/isserrano/kubernetes-ingress -w /go/src/github.com/isserrano/kubernetes-ingress/nginx-controller/
GOLANG_CONTAINER = golang:1.8
DOCKERFILE = Dockerfile

BUILD_IN_CONTAINER = 1
PUSH_TO_GCR =
GENERATE_DEFAULT_CERT_AND_KEY = 

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

certificate-and-key: 
ifeq ($(GENERATE_DEFAULT_CERT_AND_KEY),1)
	./generate_default_cert_and_key.sh
endif

container: test nginx-ingress certificate-and-key
	docker build -f $(DOCKERFILE) -t $(PREFIX):$(TAG) .

push: container
ifeq ($(PUSH_TO_GCR),1)
	gcloud docker -- push $(PREFIX):$(TAG)
else
	docker push $(PREFIX):$(TAG)
endif

osx:
ifeq ($(BUILD_IN_CONTAINER),1)
	$(DOCKER_RUN) -e CGO_ENABLED=0 -e GOOS=darwin $(GOLANG_CONTAINER) go build -a -installsuffix cgo -ldflags "-w -X main.version=${VERSION}" -o osx-nginx-ingress *.go
else
	CGO_ENABLED=0 GOOS=darwin go build -a -installsuffix cgo -ldflags "-w -X main.version=${VERSION}" -o osx-nginx-ingress *.go
endif

clean:
	rm -f nginx-ingress
