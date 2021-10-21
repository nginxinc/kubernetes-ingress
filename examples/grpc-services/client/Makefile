NAME=grpc-client-builder
TAG=0.1

compile-osx:
	docker build --build-arg GOOS=darwin -t $(NAME):$(TAG) .
	docker container create --name extract $(NAME):$(TAG)  
	docker container cp extract:/go/src/google.golang.org/grpc/examples/helloworld/greeter_client/grpc_client .  
	docker container rm -f extract
