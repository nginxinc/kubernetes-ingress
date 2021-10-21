NAME=grpc-server-ingress
TAG=0.1

build:
	docker build -t $(NAME):$(TAG) .

push: build
	docker -- push $(NAME):$(TAG)

run:
	docker run --rm -p 50051:50051 $(NAME):$(TAG)
