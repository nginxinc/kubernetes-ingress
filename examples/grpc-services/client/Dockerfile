FROM golang:1.16

ARG GOOS

RUN go get -u github.com/golang/protobuf/protoc-gen-go 
RUN git clone https://github.com/grpc/grpc-go.git $GOPATH/src/google.golang.org/grpc

WORKDIR /go/src/google.golang.org/grpc/examples/helloworld/greeter_client/
COPY main.go /go/src/google.golang.org/grpc/examples/helloworld/greeter_client/

RUN GOOS=${GOOS} go build -o grpc_client main.go 

