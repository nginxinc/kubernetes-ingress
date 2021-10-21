FROM golang:1.16
RUN go get -u github.com/golang/protobuf/protoc-gen-go 
RUN git clone https://github.com/grpc/grpc-go.git $GOPATH/src/google.golang.org/grpc
WORKDIR /go/src/google.golang.org/grpc/examples/helloworld/greeter_server/
RUN CGO_ENABLED=0 go build -a -installsuffix cgo -o grpc_server main.go 

FROM alpine:latest  
COPY --from=0 /go/src/google.golang.org/grpc/examples/helloworld/greeter_server/grpc_server /
CMD ["/grpc_server"]